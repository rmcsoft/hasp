package periph_gpio

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/conn/pin"
	"periph.io/x/periph/host/fs"
)

// Pins is all the pins exported by GPIO sysfs.
//
// Some CPU architectures have the pin numbers start at 0 and use consecutive
// pin numbers but this is not the case for all CPU architectures, some
// have gaps in the pin numbering.
//
// This global variable is initialized once at driver initialization and isn't
// mutated afterward. Do not modify it.
var Pins map[int]*Pin

// Pin represents one GPIO pin as found by sysfs.
type Pin struct {
	number int
	name   string
	root   string

	mu         sync.Mutex
	err        error     // If open() failed
	direction  direction // Cache of the last known direction
	edge       gpio.Edge // Cache of the last edge used.
	fDirection fileIO    // handle to /sys/class/gpio/gpio*/direction; never closed
	fEdge      fileIO    // handle to /sys/class/gpio/gpio*/edge; never closed
	fValue     fileIO    // handle to /sys/class/gpio/gpio*/value; never closed
	event      fs.Event  // Initialized once
	buf        [4]byte   // scratch buffer for Function(), Read() and Out()
}

// String implements conn.Resource.
func (p *Pin) String() string {
	return p.name
}

// Halt implements conn.Resource.
//
// It stops edge detection if enabled.
func (p *Pin) Halt() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.haltEdge()
}

// Name implements pin.Pin.
func (p *Pin) Name() string {
	return p.name
}

// Number implements pin.Pin.
func (p *Pin) Number() int {
	return p.number
}

// Function implements pin.Pin.
func (p *Pin) Function() string {
	return string(p.Func())
}

// Func implements pin.PinFunc.
func (p *Pin) Func() pin.Func {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.open(); err != nil {
		return pin.Func("ERR")
	}
	if _, err := seekRead(p.fDirection, p.buf[:]); err != nil {
		return pin.Func("ERR")
	}
	if p.buf[0] == 'i' && p.buf[1] == 'n' {
		p.direction = dIn
	} else if p.buf[0] == 'o' && p.buf[1] == 'u' && p.buf[2] == 't' {
		p.direction = dOut
	}
	if p.direction == dIn {
		if p.Read() {
			return gpio.IN_HIGH
		}
		return gpio.IN_LOW
	} else if p.direction == dOut {
		if p.Read() {
			return gpio.OUT_HIGH
		}
		return gpio.OUT_LOW
	}
	return pin.Func("ERR")
}

// SupportedFuncs implements pin.PinFunc.
func (p *Pin) SupportedFuncs() []pin.Func {
	return []pin.Func{gpio.IN, gpio.OUT}
}

// SetFunc implements pin.PinFunc.
func (p *Pin) SetFunc(f pin.Func) error {
	switch f {
	case gpio.IN:
		return p.In(gpio.PullNoChange, gpio.NoEdge)
	case gpio.OUT_HIGH:
		return p.Out(gpio.High)
	case gpio.OUT, gpio.OUT_LOW:
		return p.Out(gpio.Low)
	default:
		return p.wrap(errors.New("unsupported function"))
	}
}

// In implements gpio.PinIn.
func (p *Pin) In(pull gpio.Pull, edge gpio.Edge) error {
	if pull != gpio.PullNoChange && pull != gpio.Float {
		return p.wrap(errors.New("doesn't support pull-up/pull-down"))
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.direction != dIn {
		if err := p.open(); err != nil {
			return p.wrap(err)
		}
		if err := seekWrite(p.fDirection, bIn); err != nil {
			return p.wrap(err)
		}
		p.direction = dIn
	}
	// Always push none to help accumulated flush edges. This is not fool proof
	// but it seems to help.
	if p.fEdge != nil {
		if err := seekWrite(p.fEdge, bNone); err != nil {
			return p.wrap(err)
		}
	}
	// Assume that when the pin was switched, the driver doesn't recall if edge
	// triggering was enabled.
	if edge != gpio.NoEdge {
		if p.fEdge == nil {
			var err error
			p.fEdge, err = fileIOOpen(p.root+"edge", os.O_RDWR)
			if err != nil {
				return p.wrap(err)
			}
			if err = p.event.MakeEvent(p.fValue.Fd()); err != nil {
				_ = p.fEdge.Close()
				p.fEdge = nil
				return p.wrap(err)
			}
		}
		// Always reset the edge detection mode to none after starting the epoll
		// otherwise edges are not always delivered, as observed on an Allwinner A20
		// running kernel 4.14.14.
		if err := seekWrite(p.fEdge, bNone); err != nil {
			return p.wrap(err)
		}
		var b []byte
		switch edge {
		case gpio.RisingEdge:
			b = bRising
		case gpio.FallingEdge:
			b = bFalling
		case gpio.BothEdges:
			b = bBoth
		}
		if err := seekWrite(p.fEdge, b); err != nil {
			return p.wrap(err)
		}
	}
	p.edge = edge
	// This helps to remove accumulated edges but this is not 100% sufficient.
	// Most of the time the interrupts are handled promptly enough that this loop
	// flushes the accumulated interrupt.
	// Sometimes the kernel may have accumulated interrupts that haven't been
	// processed for a long time, it can easily be >300µs even on a quite idle
	// CPU. In this case, the loop below is not sufficient, since the interrupt
	// will happen afterward "out of the blue".
	if edge != gpio.NoEdge {
		p.WaitForEdge(0)
	}
	return nil
}

// Read implements gpio.PinIn.
func (p *Pin) Read() gpio.Level {
	// There's no lock here.
	if p.fValue == nil {
		return gpio.Low
	}
	if _, err := seekRead(p.fValue, p.buf[:]); err != nil {
		// Error.
		return gpio.Low
	}
	if p.buf[0] == '0' {
		return gpio.Low
	}
	if p.buf[0] == '1' {
		return gpio.High
	}
	// Error.
	return gpio.Low
}

// WaitForEdge implements gpio.PinIn.
func (p *Pin) WaitForEdge(timeout time.Duration) bool {
	// Run lockless, as the normal use is to call in a busy loop.
	var ms int
	if timeout == -1 {
		ms = -1
	} else {
		ms = int(timeout / time.Millisecond)
	}
	start := time.Now()
	for {
		if nr, err := p.event.Wait(ms); err != nil {
			return false
		} else if nr == 1 {
			return true
		}
		// A signal occurred.
		if timeout != -1 {
			ms = int((timeout - time.Since(start)) / time.Millisecond)
		}
		if ms <= 0 {
			return false
		}
	}
}

// Pull implements gpio.PinIn.
//
// It returns gpio.PullNoChange since gpio sysfs has no support for input pull
// resistor.
func (p *Pin) Pull() gpio.Pull {
	return gpio.PullNoChange
}

// DefaultPull implements gpio.PinIn.
//
// It returns gpio.PullNoChange since gpio sysfs has no support for input pull
// resistor.
func (p *Pin) DefaultPull() gpio.Pull {
	return gpio.PullNoChange
}

// Out implements gpio.PinOut.
func (p *Pin) Out(l gpio.Level) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.direction != dOut {
		if err := p.open(); err != nil {
			return p.wrap(err)
		}
		if err := p.haltEdge(); err != nil {
			return err
		}
		// "To ensure glitch free operation, values "low" and "high" may be written
		// to configure the GPIO as an output with that initial value."
		var d []byte
		if l == gpio.Low {
			d = bLow
		} else {
			d = bHigh
		}
		if err := seekWrite(p.fDirection, d); err != nil {
			return p.wrap(err)
		}
		p.direction = dOut
		return nil
	}
	if l == gpio.Low {
		p.buf[0] = '0'
	} else {
		p.buf[0] = '1'
	}
	if err := seekWrite(p.fValue, p.buf[:1]); err != nil {
		return p.wrap(err)
	}
	return nil
}

// PWM implements gpio.PinOut.
//
// This is not supported on sysfs.
func (p *Pin) PWM(gpio.Duty, physic.Frequency) error {
	return p.wrap(errors.New("pwm is not supported via sysfs"))
}

//

// open opens the gpio sysfs handle to /value and /direction.
//
// lock must be held.
func (p *Pin) open() error {
	if p.fDirection != nil || p.err != nil {
		return p.err
	}

	// There's a race condition where the file may be created but udev is still
	// running the Raspbian udev rule to make it readable to the current user.
	// It's simpler to just loop a little as if /export is accessible, it doesn't
	// make sense that gpioN/value doesn't become accessible eventually.
	timeout := 5 * time.Second
	for start := time.Now(); time.Since(start) < timeout; {
		p.fValue, p.err = fileIOOpen(p.root+"value", os.O_RDWR)
		// The virtual file creation is synchronous when writing to /export for
		// udev rule execution is asynchronous.
		if p.err == nil {
			break
		}
		if !os.IsPermission(p.err) {
			break
		}
	}
	if p.err != nil {
		return p.err
	}
	p.fDirection, p.err = fileIOOpen(p.root+"direction", os.O_RDWR)
	if p.err != nil {
		_ = p.fValue.Close()
		p.fValue = nil
	}
	return p.err
}

// haltEdge stops any on-going edge detection.
func (p *Pin) haltEdge() error {
	if p.edge != gpio.NoEdge {
		if err := seekWrite(p.fEdge, bNone); err != nil {
			return p.wrap(err)
		}
		p.edge = gpio.NoEdge
		// This is still important to remove an accumulated edge.
		p.WaitForEdge(0)
	}
	return nil
}

func (p *Pin) wrap(err error) error {
	return fmt.Errorf("sysfs-gpio (%s): %v", p, err)
}

//

type direction int

const (
	dUnknown direction = 0
	dIn      direction = 1
	dOut     direction = 2
)

var (
	bIn      = []byte("in")
	bLow     = []byte("low")
	bHigh    = []byte("high")
	bNone    = []byte("none")
	bRising  = []byte("rising")
	bFalling = []byte("falling")
	bBoth    = []byte("both")
)
