package hasp

import (
	"fmt"
	"sync/atomic"
	"time"
)

type idleState struct {
	availableAnimations []string
	animationDuration   time.Duration
	currentAnimation    int
}

// NewIdleState creates new IdleState
func NewIdleState(availableAnimations []string, animationDuration time.Duration) State {
	return &idleState{
		availableAnimations: availableAnimations,
		animationDuration:   animationDuration,
	}
}

func (s *idleState) Enter(event Event) (EventSources, error) {
	fmt.Printf("IdleState Enter\n")
	return EventSources{
			&changeAnimationEventSource{
				period: s.animationDuration,
			},
		},
		nil
}

func (s *idleState) Leave(event Event) bool {
	fmt.Printf("IdleState Leave\n")
	return true
}

func (s *idleState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	s.currentAnimation = (s.currentAnimation + 1) % len(s.availableAnimations)
	return animation
}

func (*idleState) GetSound() []int16 {
	return nil
}

type changeAnimationEventSource struct {
	period   time.Duration
	stopFlag int32
}

func (c *changeAnimationEventSource) Name() string {
	return "ChangeAnimationEventSource"
}

func (c *changeAnimationEventSource) Events() chan *Event {
	eventChan := make(chan *Event)
	go func() {
		for atomic.LoadInt32(&c.stopFlag) == 0 {
			time.Sleep(c.period)
			eventChan <- &Event{Name: StateChangedEventName}
		}
	}()
	return eventChan
}

func (c *changeAnimationEventSource) Close() {
	atomic.StoreInt32(&c.stopFlag, 1)
}
