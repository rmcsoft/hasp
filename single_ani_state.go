package hasp

import (
	"fmt"

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

func (s *singleAniState) Enter(event events.Event) (events.EventSources, error) {
	fmt.Printf("SingleAniState Enter\n")

	return nil, nil
}

func (s *singleAniState) Leave(event events.Event) bool {
	fmt.Printf("SingleAniState Leave\n")
	return true
}

func (s *singleAniState) GetAnimation() string {
	return s.Animation
}

func (s *singleAniState) GetSound() []int16 {
	return nil
}
