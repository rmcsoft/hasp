package events

import (
	"errors"
	"fmt"
)

type AwsRepliedEventData struct {
	Samples    []int16
	SampleRate int
}

const (
	AwsRepliedEventName = "AwsReplied"
)

// NewHotWordDetectedEvent creates HotWordDetectedEvent
func NewAwsRepliedEvent(samples []int16, sampleRate int) *Event {
	return &Event{
		Name: AwsRepliedEventName,
		Args: []interface{}{
			AwsRepliedEventData {samples, sampleRate},
		},
	}
}

// GetAwsRepliedEventData gets AwsRepliedEvent data
func (event *Event) GetAwsRepliedEventData() (AwsRepliedEventData, error) {
	if event.Name != AwsRepliedEventName {
		return AwsRepliedEventData {},
			fmt.Errorf("The event must be named %s", AwsRepliedEventName)
	}

	if len(event.Args) != 1 {
		return AwsRepliedEventData {},
			errors.New("Event does not data")
	}

	data, ok := event.Args[0].(AwsRepliedEventData)
	if !ok {
		return AwsRepliedEventData{},
			errors.New("Event does not contain samples")
	}

	return data, nil
}
