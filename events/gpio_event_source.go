package events

import (
	"time"

	"github.com/sirupsen/logrus"
	"periph.io/x/periph/conn/gpio"
)

const (
	GpioEventName = "GpioEvent"
)

type gpioEventSource struct {
	eventChan   chan *Event
	sensorsPins []gpio.PinIO
}

func CheckAllPins(sensorsPins []gpio.PinIO) bool {
	for _, pin := range sensorsPins {
		val := pin.Read()
		if val == gpio.Low {
			logrus.Trace("GPIO: ", pin.Name(), " is LOW")
			return false
		} else {
			logrus.Trace("GPIO: ", pin.Name(), " is HIGH")
		}
	}
	logrus.Trace("GPIO: ALL HIGH")
	return true
}

// NewSingleEventSource creates new gpioEventSource
func NewGpioEventSource(sensorsPins []gpio.PinIO) EventSource {
	es := &gpioEventSource{
		eventChan:   make(chan *Event),
		sensorsPins: sensorsPins,
	}

	if len(es.sensorsPins) > 0 {
		logrus.Trace("Starting Gpio watcher")
		go es.run()
	}
	return es
}

func (es *gpioEventSource) Name() string {
	return "GpioEventSource"
}

func (es *gpioEventSource) Events() chan *Event {
	return es.eventChan
}

func (es *gpioEventSource) Close() {
}

func (es *gpioEventSource) run() {
	defer close(es.eventChan)

	t := time.NewTicker(500 * time.Millisecond)
	for {
		if CheckAllPins(es.sensorsPins) {
			es.eventChan <- &Event{Name: GpioEventName}
			return
		}
		<-t.C
	}
}
