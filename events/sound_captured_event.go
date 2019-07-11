package events

import (
	"errors"
	"fmt"
)

type SoundCapturedEventData struct {
	Samples    []int16
	SampleRate int
}

const (
	SoundCapturedEventName = "SoundCaptured"
)

// NewHotWordDetectedEvent creates HotWordDetectedEvent
func NewSoundCapturedEvent(samples []int16, sampleRate int) *Event {
	return &Event{
		Name: SoundCapturedEventName,
		Args: []interface{}{
			SoundCapturedEventData {samples, sampleRate},
		},
	}
}

// GetHotWordDetectedEventData gets HotWordDetectedEvent data
func (event *Event) GetSoundCapturedEventData() (SoundCapturedEventData, error) {
	if event.Name != SoundCapturedEventName {
		return SoundCapturedEventData {},
			fmt.Errorf("The event must be named %s", SoundCapturedEventName)
	}

	if len(event.Args) != 1 {
		return SoundCapturedEventData {},
			errors.New("Event does not data")
	}

	data, ok := event.Args[0].(SoundCapturedEventData)
	if !ok {
		return SoundCapturedEventData{},
			errors.New("Event does not contain samples")
	}

	return data, nil
}
