package haspaws

import (
	"errors"
	"fmt"

	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"
)

type AwsRepliedEventData struct {
	RepliedSpeech *sound.AudioData
}

const (
	AwsRepliedEventName = "AwsReplied"
	AwsRepliedCallEventName = "AwsRepliedCall"
	AwsRepliedTypeEventName = "AwsRepliedType"

)

// NewAwsRepliedEvent creates RepliedEvent
func NewAwsRepliedEvent(repliedSpeech *sound.AudioData) *events.Event {
	return &events.Event{
		Name: AwsRepliedEventName,
		Args: []interface{}{
			AwsRepliedEventData{repliedSpeech},
		},
	}
}

func NewAwsRepliedEventState(repliedSpeech *sound.AudioData, name string) *events.Event {
	return &events.Event{
		Name: name,
		Args: []interface{}{
			AwsRepliedEventData{repliedSpeech},
		},
	}
}

// GetAwsRepliedEventData gets AwsRepliedEvent data
func GetAwsRepliedEventData(event *events.Event) (AwsRepliedEventData, error) {
	if event.Name != AwsRepliedEventName &&
		event.Name != AwsRepliedCallEventName &&
		event.Name != AwsRepliedTypeEventName {
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
