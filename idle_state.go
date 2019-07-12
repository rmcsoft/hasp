package hasp

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/rmcsoft/hasp/events"
)

type idleState struct {
	availableAnimations []string
	animationDuration   time.Duration
	currentAnimation    int
	hotWordDetector     *HotWordDetector
}

// NewIdleState creates new IdleState
func NewIdleState(availableAnimations []string, animationDuration time.Duration,
	hotWordDetector *HotWordDetector) State {

	return &idleState{
		availableAnimations: availableAnimations,
		animationDuration:   animationDuration,
		hotWordDetector:     hotWordDetector,
	}
}

func (s *idleState) Enter(event events.Event) (events.EventSources, error) {
	fmt.Printf("IdleState Enter\n")
	detectorEventSource, err := s.hotWordDetector.StartDetect()
	if err != nil {
		return nil, err
	}

	return events.EventSources{
		&changeAnimationEventSource{
			period: s.animationDuration,
		},
		detectorEventSource,
	}, nil
}

func (s *idleState) Leave(event events.Event) bool {
	fmt.Printf("IdleState Leave\n")
	return true
}

func (s *idleState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	s.currentAnimation = (s.currentAnimation + 1) % len(s.availableAnimations)
	return animation
}

func (s *idleState) GetSound() []int16 {
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
