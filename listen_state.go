package hasp

import (
	"fmt"

	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"
)

type listensState struct {
	availableAnimations []string
	currentAnimation    int
	detector            *sound.HotWordDetector
}

// NewListensState creates new ListensState
func NewListensState(availableAnimations []string, detector *sound.HotWordDetector) State {
	return &listensState{
		availableAnimations: availableAnimations,
		detector:            detector,
	}
}

func (s *listensState) Enter(ctx CharacterCtx, event events.Event) (events.EventSources, error) {
	fmt.Printf("ListensState Enter\n")

	soundCapturerEventSource, err := s.detector.StartSoundCapture()
	if err != nil {
		panic(err)
	}

	return events.EventSources{
		soundCapturerEventSource,
	}, nil
}

func (s *listensState) Leave(ctx CharacterCtx, event events.Event) bool {
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
