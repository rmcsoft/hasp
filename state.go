package hasp

import (
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"
)

// State defines an interface for states
type State interface {
	Enter(ctx CharacterCtx, event events.Event) (events.EventSources, error)
	Leave(ctx CharacterCtx, event events.Event) bool

	GetAnimation() string
	GetSound() *sound.AudioData
}

// States is set of states.
type States = map[string]State
