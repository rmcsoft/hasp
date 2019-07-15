package hasp

import hasp "github.com/rmcsoft/hasp/events"

// State defines an interface for states
type State interface {
	Enter(ctx CharacterCtx, event hasp.Event) (hasp.EventSources, error)
	Leave(ctx CharacterCtx, event hasp.Event) bool

	GetAnimation() string
	GetSound() []int16
}

// States is set of states.
type States = map[string]State
