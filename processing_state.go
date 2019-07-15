package hasp

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/lexruntimeservice"
	"github.com/rmcsoft/hasp/events"
)

type processingState struct {
	availableAnimations []string
	currentAnimation    int
	lrs                 *lexruntimeservice.LexRuntimeService
}

// NewProcessingState creates new ProcessingState
func NewProcessingState(availableAnimations []string, lrs *lexruntimeservice.LexRuntimeService) State {
	return &processingState{
		availableAnimations: availableAnimations,
		lrs:                 lrs,
	}
}

func (s *processingState) Enter(event events.Event) (events.EventSources, error) {
	fmt.Printf("ProcessingState Enter\n")

	data, _ := event.GetSoundCapturedEventData()
	soundCapturer, err := NewLexEventSource(s.lrs, data)
	if err != nil {
		panic(err)
	}

	return events.EventSources{
		soundCapturer,
	}, nil
}

func (s *processingState) Leave(event events.Event) bool {
	fmt.Printf("ProcessingState Leave\n")
	return true
}

func (s *processingState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	return animation
}

func (*processingState) GetSound() []int16 {
	return nil
}
