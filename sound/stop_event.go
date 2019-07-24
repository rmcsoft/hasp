package sound

import (
	"errors"
	"fmt"

	"github.com/rmcsoft/hasp/events"
)

type StopEventData struct {
	StopSpeach *AudioData
}

const (
	StopEventName = "Stop"
)

func NewStopEvent(stopSpeach *AudioData) *events.Event {
	return &events.Event{
		Name: StopEventName,
		Args: []interface{}{
			StopEventData{stopSpeach},
		},
	}
}

func GetStopEventData(event *events.Event) (StopEventData, error) {
	if event.Name != StopEventName {
		return StopEventData{},
			fmt.Errorf("The event must be named %s", StopEventName)
	}

	if len(event.Args) != 1 {
		return StopEventData{},
			errors.New("Event does not data")
	}

	data, ok := event.Args[0].(StopEventData)
	if !ok {
		return StopEventData{},
			errors.New("Event does not contain samples")
	}

	return data, nil
}
