package hasp

import (
	"fmt"
	"os"

	"github.com/looplab/fsm"
	"github.com/rmcsoft/chanim"
)

// EventDescs is a shorthand for defining the transition map
type EventDescs = []fsm.EventDesc

// Character is animated character
type Character struct {
	states   States
	animator *chanim.Animator
	fsm      *fsm.FSM

	eventSourceMultiplexer EventSourceMultiplexer
	stateEventSources      []IDEventSource
}

// NewCharacter creates new a Character
func NewCharacter(initStateName string, states States, eventDescs EventDescs, eventSources EventSources, animator *chanim.Animator) (*Character, error) {
	c := &Character{
		states:   states,
		animator: animator,
	}

	c.fsm = fsm.NewFSM(
		initStateName,
		eventDescs,
		fsm.Callbacks{
			"enter_state": func(e *fsm.Event) {
				c.enterStateCallbacks(e)
			},
			"leave_state": func(e *fsm.Event) {
				c.leaveStateCallback(e)
			},
			"after_event": func(e *fsm.Event) {
				c.updateCharacter(e, false)
			},
		},
	)

	for _, eventSource := range eventSources {
		c.eventSourceMultiplexer.AddEventSource(eventSource)
	}

	return c, nil
}

// Run starting point for the character
func (c *Character) Run() error {
	if err := c.start(); err != nil {
		return err
	}

	for event := c.eventSourceMultiplexer.NextEvent(); event != nil; {
		err := c.fsm.Event(event.Name, event.Args...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
		}
	}

	return nil
}

func (c *Character) start() error {
	initStateName := c.fsm.Current()

	startEvent := fsm.Event{
		Dst: initStateName,
	}

	c.enterStateCallbacks(&startEvent)
	if startEvent.Err != nil {
		return startEvent.Err
	}

	c.updateCharacter(&startEvent, true)
	if startEvent.Err != nil {
		return startEvent.Err
	}

	return nil
}

func (c *Character) enterStateCallbacks(e *fsm.Event) {
	nextState, ok := c.states[e.Dst]
	if !ok {
		e.Cancel(fmt.Errorf("Can't find state with name '%s'", e.Dst))
		return
	}

	eventSources, err := nextState.Enter(Event{e.Event, e.Args})
	if err != nil {
		e.Cancel(err)
	}

	if eventSources != nil {
		stateEventSources := make([]IDEventSource, 0, len(eventSources))
		for _, eventSource := range eventSources {
			idEventSource := c.eventSourceMultiplexer.AddEventSource(eventSource)
			stateEventSources = append(stateEventSources, idEventSource)
		}
		c.stateEventSources = stateEventSources
	}
}

func (c *Character) leaveStateCallback(e *fsm.Event) {
	if predState, ok := c.states[e.Src]; ok {
		if !predState.Leave(Event{e.Event, e.Args}) {
			e.Cancel()
			return
		}

		for _, idEventSource := range c.stateEventSources {
			c.eventSourceMultiplexer.RemoveEventSource(idEventSource)
		}
		c.stateEventSources = nil
	}
}

func (c *Character) updateCharacter(e *fsm.Event, start bool) {
	stateName := c.fsm.Current()
	if state, ok := c.states[stateName]; ok {
		animation := state.GetAnimation()
		var err error
		if start {
			err = c.animator.Start(animation)
		} else {
			err = c.animator.ChangeAnimation(animation)
		}

		if err != nil {
			e.Cancel(err)
		}
	}
}
