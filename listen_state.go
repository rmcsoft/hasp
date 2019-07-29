package hasp

import (
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"
)

type listensState struct {
	availableAnimations []string
	currentAnimation    int
	detector            *sound.HotWordDetector
	soundPlayer         *sound.SoundPlayer
	enterSoundData      *sound.AudioData
	exitSoundData       *sound.AudioData
}

// NewListensState creates new ListensState
func NewListensState(availableAnimations []string, detector *sound.HotWordDetector,
	soundPlayer *sound.SoundPlayer, enterSoundData *sound.AudioData, exitSoundData *sound.AudioData) State {
	return &listensState{
		availableAnimations: availableAnimations,
		detector:            detector,
		soundPlayer:         soundPlayer,
		enterSoundData:      enterSoundData,
		exitSoundData:       exitSoundData,
	}
}

func (s *listensState) Enter(ctx CharacterCtx, event events.Event) (events.EventSources, error) {
	s.soundPlayer.PlaySync(s.enterSoundData)

	soundCapturerEventSource, err := s.detector.StartSoundCapture()
	if err != nil {
		panic(err)
	}

	return events.EventSources{
		soundCapturerEventSource,
	}, nil
}

func (s *listensState) Leave(ctx CharacterCtx, event events.Event) bool {
	s.soundPlayer.PlaySync(s.exitSoundData)
	return true
}

func (s *listensState) GetAnimation() string {
	animation := s.availableAnimations[s.currentAnimation]
	return animation
}

func (*listensState) GetSound() *sound.AudioData {
	return nil
}
