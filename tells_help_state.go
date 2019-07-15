package hasp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"

	hasp "github.com/rmcsoft/hasp/events"
)

type tellsHelpState struct {
	availableAnimations []string
	currentAnimation    int
	welcomeSpeech       []int16
}

// NewTellsHelpState creates new IdleState
func NewTellsHelpState(availableAnimations []string, welcomeSpeech string) State {
	content, err := ioutil.ReadFile(welcomeSpeech)
	if err != nil {
		log.Fatal(err)
	}

	r := bytes.NewReader(content)
	frames := make([]int16, len(content)/2)
	binary.Read(r, binary.LittleEndian, &frames)

	return &tellsHelpState{
		availableAnimations: availableAnimations,
		welcomeSpeech:       frames,
	}
}

func (s *tellsHelpState) Enter(ctx CharacterCtx, event hasp.Event) (hasp.EventSources, error) {
	fmt.Printf("TellsHelpState Enter\n")
	return nil, nil
}

func (s *tellsHelpState) Leave(ctx CharacterCtx, event hasp.Event) bool {
	fmt.Printf("TellsHelpState Leave\n")
	return true
}

func (s *tellsHelpState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	return animation
}

func (s *tellsHelpState) GetSound() []int16 {
	return s.welcomeSpeech
}
