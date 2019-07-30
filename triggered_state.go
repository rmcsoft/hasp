package hasp

import (
	"time"

	atmel "github.com/rmcsoft/hasp/atmel/periph_gpio"
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"
	"periph.io/x/periph/conn/gpio"
)

type triggeredState struct {
	availableAnimation string
	hotWordDetector    *sound.HotWordDetector
	sensorsPins        []gpio.PinIO
	waitTime           time.Duration
}

// NewTriggeredState creates new TriggeredState
func NewTriggeredState(availableAnimation string,
	hotWordDetector *sound.HotWordDetector, sensorsPins atmel.AtmelGpioPins,
	waitTime time.Duration) State {

	atmelPins := loadAtmelPeriph(sensorsPins)

	return &triggeredState{
		availableAnimation: availableAnimation,
		hotWordDetector:    hotWordDetector,
		sensorsPins:        atmelPins,
		waitTime:           waitTime,
	}
}

func (s *triggeredState) Enter(ctx CharacterCtx, event events.Event) (events.EventSources, error) {
	detectorEventSource, err := s.hotWordDetector.StartDetect()
	if err != nil {
		return nil, err
	}

	sources := events.EventSources{
		detectorEventSource,
	}

	if s.waitTime > 0 {
		src := events.NewSingleEventSource("WaitTimer",
			func() *events.Event {
				time.Sleep(s.waitTime)
				if events.CheckAllPins(s.sensorsPins) {
					return &events.Event{Name: events.StateFullHelpName }
				} else {
					return &events.Event{Name: events.StateWaitTimeoutName }
				}
			})
		sources = append(sources, src)
	}

	return sources, nil
}

func (s *triggeredState) Leave(ctx CharacterCtx, event events.Event) bool {
	return true
}

func (s *triggeredState) GetAnimation() string {
	return s.availableAnimation
}

func (s *triggeredState) GetSound() *sound.AudioData {
	return nil
}
