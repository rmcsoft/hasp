package hasp

import (
	"sync/atomic"
	"time"

	atmel "github.com/rmcsoft/hasp/atmel/periph_gpio"
	"github.com/sirupsen/logrus"

	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"
	"periph.io/x/periph"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
)

type idleState struct {
	availableAnimations []string
	animationDuration   time.Duration
	currentAnimation    int
	hotWordDetector     *sound.HotWordDetector
	sensorsPins         []gpio.PinIO
}

func atmelLoaded(state *periph.State) bool {
	for _, a := range state.Loaded {
		if atmel.AtmelGpioDriverName == a.String() {
			return true
		}
	}
	return false
}

func loadAtmelPeriph(sensorsPins atmel.AtmelGpioPins) []gpio.PinIO {
	if err := periph.Register(atmel.AtmelGpioDriver{
		Pins: sensorsPins,
	}); err != nil {
		logrus.Error(err)
		logrus.Info("Will not use GPIO sensors detection")
		return nil
	}

	// Initialize normally. Your driver will be loaded:
	state, err := host.Init()
	if err != nil {
		logrus.Error(err)
		return nil
	}

	if !atmelLoaded(state) {
		logrus.Info("Will not use GPIO sensors detection")
		return nil
	}

	atmelPins := make([]gpio.PinIO, len(sensorsPins))
	for i := 0; i < len(sensorsPins); i++ {
		atmelPins[i] = gpioreg.ByName(sensorsPins[i].Name)
		if atmelPins[i] == nil {
			logrus.Error("Failed to open pin ", sensorsPins[i].Name)
		} else {
			err = atmelPins[i].In(gpio.PullNoChange, gpio.NoEdge)
			if err != nil {
				logrus.Error(err)
			} else {
				logrus.Debug("GPIO ", sensorsPins[i].Name, " is ready")
			}
		}
	}
	return atmelPins
}

// NewIdleState creates new IdleState
func NewIdleState(availableAnimations []string, animationDuration time.Duration,
	hotWordDetector *sound.HotWordDetector, sensorsPins atmel.AtmelGpioPins) State {

	atmelPins := loadAtmelPeriph(sensorsPins)

	return &idleState{
		availableAnimations: availableAnimations,
		animationDuration:   animationDuration,
		hotWordDetector:     hotWordDetector,
		sensorsPins:         atmelPins,
	}
}

func (s *idleState) Enter(ctx CharacterCtx, event events.Event) (events.EventSources, error) {
	detectorEventSource, err := s.hotWordDetector.StartDetect()
	if err != nil {
		return nil, err
	}

	return events.EventSources{
		events.NewGpioEventSource(s.sensorsPins),
		&changeAnimationEventSource{
			period: s.animationDuration,
		},
		detectorEventSource,
	}, nil
}

func (s *idleState) Leave(ctx CharacterCtx, event events.Event) bool {
	return true
}

func (s *idleState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	s.currentAnimation = (s.currentAnimation + 1) % len(s.availableAnimations)
	return animation
}

func (s *idleState) GetSound() *sound.AudioData {
	return nil
}

type changeAnimationEventSource struct {
	period   time.Duration
	stopFlag int32
}

func (c *changeAnimationEventSource) Name() string {
	return "ChangeAnimationEventSource"
}

func (c *changeAnimationEventSource) Events() chan *events.Event {
	eventChan := make(chan *events.Event)
	go func() {
		for atomic.LoadInt32(&c.stopFlag) == 0 {
			time.Sleep(c.period)
			eventChan <- &events.Event{Name: events.StateChangedEventName}
		}
	}()
	return eventChan
}

func (c *changeAnimationEventSource) Close() {
	atomic.StoreInt32(&c.stopFlag, 1)
}
