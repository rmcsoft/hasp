package hasp

import (
	"fmt"

	"github.com/rmcsoft/hasp/events"
)

type tellsState struct {
	availableAnimations []string
	currentAnimation    int
	data                []int16
}

// NewAwakeState creates new IdleState
func NewTellsState(availableAnimations []string) State {
	return &tellsState{
		availableAnimations: availableAnimations,
	}
}

func (s *tellsState) Enter(event events.Event) (events.EventSources, error) {
	fmt.Printf("TellsState Enter\n")
	data, _ := event.GetSoundCapturedEventData()
	s.data = data.Samples
	return nil, nil
}

func (s *tellsState) Leave(event events.Event) bool {
	fmt.Printf("TellsState Leave\n")
	return true
}

func (s *tellsState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	return animation
}

func (s *tellsState) GetSound() []int16 {
	return s.data
}
