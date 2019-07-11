package hasp

import (
	"fmt"

	"github.com/rmcsoft/hasp/events"
)

type listensState struct {
	availableAnimations []string
	currentAnimation    int
	captureDevice       string
}

func NewListensState(availableAnimations []string, captureDevice string) State {
	return &listensState{
		availableAnimations: availableAnimations,
		captureDevice:       captureDevice,
	}
}

func (s *listensState) Enter(event events.Event) (events.EventSources, error) {
	fmt.Printf("ListensState Enter\n")

	soundCapturer, err := NewSoundCapturer(s.captureDevice)
	if err != nil {
		panic(err)
	}

	return events.EventSources{
		soundCapturer,
	}, nil
}

func (s *listensState) Leave(event events.Event) bool {
	fmt.Printf("ListensState Leave\n")
	return true
}

func (s *listensState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	return animation
}

func (*listensState) GetSound() []int16 {
	return nil
}