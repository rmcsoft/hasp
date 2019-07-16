package sound

import (
	"errors"
	"fmt"

	"github.com/rmcsoft/hasp/events"
)

// HotWordDetectedEventData is the HotWordDetectedEvent data
type HotWordDetectedEventData struct {
	AudioData *AudioData
}

const (
	// HotWordDetectedEventName is the event name for keyword detection
	HotWordDetectedEventName = "HotWordDetected"
	HotWordWithDataDetectedEventName = "HotWordWithDataDetected"
)

// NewHotWordDetectedEvent creates HotWordDetectedEvent
func NewHotWordDetectedEvent(audioData *AudioData) *events.Event {
	typeName := HotWordWithDataDetectedEventName
	if len(audioData.samples) == 0 {
		typeName = HotWordDetectedEventName
	}
	return &events.Event{
		Name: typeName,
		Args: []interface{}{
			SoundCapturedEventData{audioData},
		},
	}
}

// GetHotWordDetectedEventData gets HotWordDetectedEvent data
func GetHotWordDetectedEventData(event *events.Event) (HotWordDetectedEventData, error) {
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