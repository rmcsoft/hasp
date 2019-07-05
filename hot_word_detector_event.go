package hasp

import (
	"errors"
	"fmt"
)

// HotWordDetectedEventData is the HotWordDetectedEvent data
type HotWordDetectedEventData struct {
	Samples    []int16
	SampleRate int
}

const (
	// HotWordDetectedEventName is the event name for keyword detection
	HotWordDetectedEventName = "HotWordDetected"
)

// NewHotWordDetectedEvent creates HotWordDetectedEvent
func NewHotWordDetectedEvent(samples []int16, sampleRate int) *Event {
	return &Event{
		Name: HotWordDetectedEventName,
		Args: []interface{}{
			HotWordDetectedEventData{samples, sampleRate},
		},
	}
}

// GetHotWordDetectedEventData gets HotWordDetectedEvent data
func (event *Event) GetHotWordDetectedEventData() (HotWordDetectedEventData, error) {
	if event.Name != HotWordDetectedEventName {
		return HotWordDetectedEventData{},
			fmt.Errorf("The event must be named %s", HotWordDetectedEventName)
	}

	if len(event.Args) != 1 {
		return HotWordDetectedEventData{},
			errors.New("Event does not data")
	}

	data, ok := event.Args[0].(HotWordDetectedEventData)
	if !ok {
		return HotWordDetectedEventData{},
			errors.New("Event does not contain samples")
	}

	return data, nil
}
