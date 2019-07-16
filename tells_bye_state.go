package hasp

import (
	"fmt"

	"github.com/rmcsoft/hasp/haspaws"
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
	data, _ := haspaws.GetStopEventData(&event)
	s.byeSpeech = data.StopSpeach
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
