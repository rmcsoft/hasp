package hasp

import (
	"github.com/rmcsoft/hasp/haspaws"
	"github.com/rmcsoft/hasp/sound"

	"github.com/rmcsoft/hasp/events"
)

type tellsState struct {
	availableAnimations []string
	currentAnimation    int
	speech              *sound.AudioData
}

// NewTellsState creates new IdleState
func NewTellsState(availableAnimations []string) State {
	return &tellsState{
		availableAnimations: availableAnimations,
	}
}

func (s *tellsState) Enter(ctx CharacterCtx, event events.Event) (events.EventSources, error) {
	data, _ := haspaws.GetAwsRepliedEventData(&event)
	s.speech = data.RepliedSpeech
	if s.speech == nil || len(s.speech.Samples()) == 0 {
		return events.EventSources{events.NewSingleEventSource(sound.SoundPlayedEventName, func() *events.Event {
			return &events.Event{Name: sound.SoundPlayedEventName} }) }, nil
	}
	return nil, nil
}

func (s *tellsState) Leave(ctx CharacterCtx, event events.Event) bool {
	return true
}

func (s *tellsState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	return animation
}

func (s *tellsState) GetSound() *sound.AudioData {
	return s.speech
}
