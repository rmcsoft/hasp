package hasp

import (
	"fmt"

	"github.com/rmcsoft/hasp/sound"

	"github.com/rmcsoft/hasp/events"
)

type singleAniState struct {
	Animation string
}

func NewSingleAniState(animation string) State {
	return &singleAniState{
		Animation: animation,
	}
}

func (s *singleAniState) Enter(ctx CharacterCtx, event events.Event) (events.EventSources, error) {
	fmt.Printf("SingleAniState Enter\n")

	return events.EventSources{events.NewSingleEventSource(events.StateGoIdleName, func() *events.Event {
		return &events.Event{Name: events.StateGoIdleName}
	})}, nil
}

func (s *singleAniState) Leave(ctx CharacterCtx, event events.Event) bool {
	fmt.Printf("SingleAniState Leave\n")
	return true
}

func (s *singleAniState) GetAnimation() string {
	return s.Animation
}

func (s *singleAniState) GetSound() *sound.AudioData {
	return nil
}
