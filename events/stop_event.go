package events

import (
	"errors"
	"fmt"
)

type StopEventData struct {
	Samples    []int16
	SampleRate int
}

const (
	StopEventName = "Stop"
)

func NewStopEvent(samples []int16, sampleRate int) *Event {
	return &Event{
		Name: StopEventName,
		Args: []interface{}{
			StopEventData{samples, sampleRate},
		},
	}
}

func (event *Event) GetStopEventData() (StopEventData, error) {
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
