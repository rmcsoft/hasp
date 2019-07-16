package hasp

import (
	"fmt"

	"github.com/rmcsoft/hasp/sound"

	"github.com/rmcsoft/hasp/events"
)

type tellsByeState struct {
	availableAnimations []string
	currentAnimation    int
	byeSpeech           *sound.AudioData
}

func NewTellsByeState(availableAnimations []string) State {
	return &tellsByeState{
		availableAnimations: availableAnimations,
	}
}

func (s *tellsByeState) Enter(ctx CharacterCtx, event events.Event) (events.EventSources, error) {
	fmt.Printf("TellsByeState Enter\n")
	data, _ := event.GetStopEventData()
	s.byeSpeech = sound.NewMonoS16LEFromInt16(data.SampleRate, data.Samples)
	return nil, nil
}

func (s *tellsByeState) Leave(ctx CharacterCtx, event events.Event) bool {
	delete(ctx, CtxUserId)
	fmt.Printf("TellsByeState Leave\n")
	return true
}

func (s *tellsByeState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	return animation
}

func (s *tellsByeState) GetSound() *sound.AudioData {
	return s.byeSpeech
}
