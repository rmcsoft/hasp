package hasp

import (
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
	s.byeSpeech = nil
	if len(event.Args) > 0 {
		data, _ := haspaws.GetStopEventData(&event)
		s.byeSpeech = data.StopSpeach
	}

	if s.byeSpeech == nil {
		return events.EventSources{events.NewSingleEventSource(sound.SoundPlayedEventName, func() *events.Event {
			return &events.Event{Name: sound.SoundPlayedEventName}
		})}, nil
	}
	return nil, nil
}

func (s *tellsByeState) Leave(ctx CharacterCtx, event events.Event) bool {
	delete(ctx, CtxUserId)
	return true
}

func (s *tellsByeState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	return animation
}

func (s *tellsByeState) GetSound() *sound.AudioData {
	return s.byeSpeech
}
