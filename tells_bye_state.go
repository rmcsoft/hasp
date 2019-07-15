package hasp

import (
	"fmt"

	"github.com/rmcsoft/hasp/events"
)

type tellsByeState struct {
	availableAnimations []string
	currentAnimation    int
	data                []int16
}

func NewTellsByeState(availableAnimations []string) State {
	return &tellsByeState{
		availableAnimations: availableAnimations,
	}
}

func (s *tellsByeState) Enter(event events.Event) (events.EventSources, error) {
	fmt.Printf("TellsByeState Enter\n")
	data, _ := event.GetStopEventData()
	s.data = data.Samples
	return nil, nil
}

func (s *tellsByeState) Leave(event events.Event) bool {
	fmt.Printf("TellsByeState Leave\n")
	return true
}

func (s *tellsByeState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	return animation
}

func (s *tellsByeState) GetSound() []int16 {
	return s.data
}