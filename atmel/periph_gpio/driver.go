package periph_gpio

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host/fs"
)

type AtmelGpioPin struct {
	Number int
	Name   string
}

type AtmelGpioPins []AtmelGpioPin

type AtmelGpioDriver struct {
	Pins AtmelGpioPins
}

const (
	AtmelGpioDriverName = "Atmel-GPIO-driver"
)

func (d AtmelGpioDriver) String() string          { return AtmelGpioDriverName }
func (d AtmelGpioDriver) Prerequisites() []string { return nil }
func (d AtmelGpioDriver) After() []string         { return nil }

func (d AtmelGpioDriver) Init() (bool, error) {
	f, err := fs.Open("/sys/class/gpio/export", os.O_WRONLY)
	if os.IsPermission(err) {
		return true, fmt.Errorf("need more access, try as root or setup udev rules: %v", err)
	}
	defer f.Close()

	// There are hosts that use non-continuous pin numbering so use a map instead
	// of an array.
	Pins = map[int]*Pin{}
	for _, item := range d.Pins {
		f.WriteString(fmt.Sprintf("%d", item.Number))

		p := &Pin{
			number: item.Number,
			name:   fmt.Sprintf("%v", item.Name),
			root:   fmt.Sprintf("/sys/class/gpio/%v/", item.Name),
		}
		//Pins[i] = p
		if err := gpioreg.Register(p); err != nil {
			return false, err
		}
		// If there is a CPU memory mapped gpio pin with the same number, the
		// driver has to unregister this pin and map its own after.
		if err := gpioreg.RegisterAlias(strconv.Itoa(item.Number), p.name); err != nil {
			//return false, err
			logrus.Info("Adding alias failed: ", strconv.Itoa(item.Number), "to", p.name)
		}
	}
	return true, err
}
