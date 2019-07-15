package sound

import "github.com/rmcsoft/hasp/events"

type SoundEmptyEventData struct {
}

const (
	SoundEmptyEventName = "SoundEmpty"
)

// NewSoundEmptyEvent creates SoundEmptyEvent
func NewSoundEmptyEvent() *events.Event {
	return &events.Event{
		Name: SoundEmptyEventName,
		Args: []interface{}{
			SoundEmptyEventData{},
		},
	}
}
