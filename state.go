package hasp

// State defines an interface for states
type State interface {
	Enter(event Event) (EventSources, error)
	Leave(event Event) bool

	GetAnimation() string
	GetSound() []int16
}

// States is set of states.
type States = map[string]State
