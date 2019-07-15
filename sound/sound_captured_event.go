package sound

import (
	"errors"
	"fmt"

	"github.com/rmcsoft/hasp/events"
)

type SoundCapturedEventData struct {
	AudioData *AudioData
}

const (
	SoundCapturedEventName = "SoundCaptured"
)

// NewSoundCapturedEvent creates HotWordDetectedEvent
func NewSoundCapturedEvent(audioData *AudioData) *events.Event {
	return &events.Event{
		Name: SoundCapturedEventName,
		Args: []interface{}{
			SoundCapturedEventData{audioData},
		},
	}
}

// GetSoundCapturedEventData gets HotWordDetectedEvent data
func GetSoundCapturedEventData(event *events.Event) (SoundCapturedEventData, error) {
	if event.Name != SoundCapturedEventName {
		return SoundCapturedEventData{},
			fmt.Errorf("The event must be named %s", SoundCapturedEventName)
	}

	if len(event.Args) != 1 {
		return SoundCapturedEventData{},
			errors.New("Event does not data")
	}

	data, ok := event.Args[0].(SoundCapturedEventData)
	if !ok {
		return SoundCapturedEventData{},
			errors.New("Event does not contain samples")
	}

	return data, nil
}
