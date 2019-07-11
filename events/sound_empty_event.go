package events

type SoundEmptyEventData struct {
}

const (
	SoundEmptyEventName = "SoundEmpty"
)

// NewSoundEmptyEvent creates SoundEmptyEvent
func NewSoundEmptyEvent() *Event {
	return &Event{
		Name: SoundEmptyEventName,
		Args: []interface{}{
			SoundEmptyEventData { },
		},
	}
}

