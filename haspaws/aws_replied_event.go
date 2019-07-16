package haspaws

import (
	"errors"
	"fmt"

	"github.com/rmcsoft/hasp/events"
)

type AwsRepliedEventData struct {
	Samples    []int16
	SampleRate int
}

const (
	AwsRepliedEventName = "AwsReplied"
)

// NewAwsRepliedEvent creates HotWordDetectedEvent
func NewAwsRepliedEvent(samples []int16, sampleRate int) *events.Event {
	return &events.Event{
		Name: AwsRepliedEventName,
		Args: []interface{}{
			AwsRepliedEventData{samples, sampleRate},
		},
	}
}

// GetAwsRepliedEventData gets AwsRepliedEvent data
func GetAwsRepliedEventData(event *events.Event) (AwsRepliedEventData, error) {
	if event.Name != AwsRepliedEventName {
		return AwsRepliedEventData{},
			fmt.Errorf("The event must be named %s", AwsRepliedEventName)
	}

	if len(event.Args) != 1 {
		return AwsRepliedEventData{},
			errors.New("Event does not data")
	}

	data, ok := event.Args[0].(AwsRepliedEventData)
	if !ok {
		return AwsRepliedEventData{},
			errors.New("Event does not contain samples")
	}

	return data, nil
}
