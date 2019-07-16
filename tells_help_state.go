package hasp

import (
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"
)

type tellsHelpState struct {
	availableAnimations []string
	currentAnimation    int
	welcomeSpeech       *sound.AudioData
}

// NewTellsHelpState creates new IdleState
func NewTellsHelpState(availableAnimations []string, welcomeSpeech *sound.AudioData) State {
	return &tellsHelpState{
		availableAnimations: availableAnimations,
		welcomeSpeech:       welcomeSpeech,
	}
}

func (s *tellsHelpState) Enter(ctx CharacterCtx, event events.Event) (events.EventSources, error) {
	return nil, nil
}

func (s *tellsHelpState) Leave(ctx CharacterCtx, event events.Event) bool {
	return true
}

func (s *tellsHelpState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	return animation
}

func (s *tellsHelpState) GetSound() *sound.AudioData {
	return s.welcomeSpeech
}
