package hasp

type inactiveState struct {
	animation string
}

// NewInactiveState creates inactiveState
func NewInactiveState(animation string) State {
	return &inactiveState{
		animation: animation,
	}
}

func (s *inactiveState) Enter(event Event) (EventSources, error) {
	return nil, nil
}

func (s *inactiveState) Leave(event Event) bool {
	return true
}

func (s *inactiveState) GetAnimation() string {
	return s.animation
}

func (s *inactiveState) GetSound() []uint16 {
	return nil
}
