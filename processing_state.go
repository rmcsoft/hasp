package hasp

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/lexruntimeservice"
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/haspaws"
	"github.com/rmcsoft/hasp/sound"
	"github.com/twinj/uuid"
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

func (s *processingState) Enter(ctx CharacterCtx, event events.Event) (events.EventSources, error) {
	fmt.Printf("ProcessingState Enter\n")

	data, _ := sound.GetSoundCapturedEventData(&event)
	userId, ok := ctx[CtxUserId]
	if !ok {
		u := uuid.NewV4()
		ctx[CtxUserId] = u.String()
		userId = ctx[CtxUserId]
	}
	lexResponseSource, err := haspaws.NewLexEventSource(s.lrs, data.AudioData, userId.(string))
	if err != nil {
		panic(err)
	}

	return events.EventSources{
		lexResponseSource,
	}, nil
}

func (s *processingState) Leave(ctx CharacterCtx, event events.Event) bool {
	fmt.Printf("ProcessingState Leave\n")
	return true
}

func (s *processingState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	return animation
}

func (*processingState) GetSound() *sound.AudioData {
	return nil
}
