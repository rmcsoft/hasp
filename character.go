package hasp

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/rmcsoft/chanim"
	"github.com/rmcsoft/hasp/events"
	"github.com/rmcsoft/hasp/sound"

	"github.com/looplab/fsm"
)

// EventDesc represents an event when initializing the FSM
type EventDesc = fsm.EventDesc

// EventDescs is a shorthand for defining the transition map
type EventDescs = []EventDesc

type CharacterCtx = map[string]interface{}

const (
	CtxUserId = "UserId"
)

// Character is animated character
type Character struct {
	states      States
	animator    *chanim.Animator
	soundPlayer *sound.SoundPlayer
	ctx         CharacterCtx
	fsm         *fsm.FSM

	eventSourceMultiplexer *events.EventSourceMultiplexer

	// Event sources that are added when entering the state
	// and removed when exiting the state
	stateEventSources []events.IDEventSource

	// Event sources that are replaced with each StateChangedEvent
	stateChangedEventSources []events.IDEventSource
}

// NewCharacter creates new a Character
func NewCharacter(
	initStateName string,
	states States,
	eventDescs EventDescs,
	eventSources events.EventSources,
	animator *chanim.Animator,
	soundPlayer *sound.SoundPlayer) (*Character, error) {

	c := &Character{
		states:                 states,
		animator:               animator,
		eventSourceMultiplexer: events.NewEventSourceMultiplexer(),
		soundPlayer:            soundPlayer,
		ctx:                    make(CharacterCtx),
	}

	// In any of the states, the StateChanged event should lead to updating
	// the animation and sound without going to another state.
	for stateName := range states {
		eventDescs = append(eventDescs, EventDesc{
			Name: events.StateChangedEventName,
			Src:  []string{stateName},
			Dst:  stateName,
		})
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

// Visualize outputs a visualization of a character FSM in Graphviz format
func (c *Character) Visualize() string {
	return fsm.Visualize(c.fsm)
}

func (c *Character) SetDebug(val bool) {
	c.ctx["Debug"] = val
}

func isNoTransitionError(err error) bool {
	_, ok := err.(fsm.NoTransitionError)
	return ok
}

// Run starting point for the character
func (c *Character) Run() error {
	if err := c.start(); err != nil {
		return err
	}

	for event := c.eventSourceMultiplexer.NextEvent(); event != nil; {

		err := c.fsm.Event(event.Name, event.Args...)
		if err != nil && !isNoTransitionError(err) {
			log.Errorf("%v\n", err)
		}

		event = c.eventSourceMultiplexer.NextEvent()
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
	log.Infof("Entry to '%s' state", e.Dst)

	nextState, ok := c.states[e.Dst]
	if !ok {
		e.Cancel(fmt.Errorf("Can't find state with name '%s'", e.Dst))
		return
	}

	eventSources, err := nextState.Enter(c.ctx, events.Event{Name: e.Event, Args: e.Args})
	if err != nil {
		e.Cancel(err)
	}

	if eventSources != nil {
		stateEventSources := make([]events.IDEventSource, 0, len(eventSources))
		for _, eventSource := range eventSources {
			idEventSource := c.addEventSource(eventSource)
			stateEventSources = append(stateEventSources, idEventSource)
		}
		c.stateEventSources = stateEventSources
	}
}

func (c *Character) leaveStateCallback(e *fsm.Event) {
	log.Infof("Leave from '%s' state", e.Src)

	if predState, ok := c.states[e.Src]; ok {
		if !predState.Leave(c.ctx, events.Event{Name: e.Event, Args: e.Args}) {
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
	// Remove older EventSources
	for _, idEventSource := range c.stateChangedEventSources {
		c.removeEventSource(idEventSource)
	}
	c.stateChangedEventSources = nil

	stateName := c.fsm.Current()
	if state, ok := c.states[stateName]; ok {
		var err error

		animation := state.GetAnimation()
		if start {
			err = c.animator.Start(animation)
		} else {
			err = c.animator.ChangeAnimation(animation)
		}
		if err != nil {
			e.Cancel(err)
			return
		}

		sound := state.GetSound()
		if sound != nil {
			eventSources, err := c.soundPlayer.Play(sound)
			if err != nil {
				e.Cancel(err)
				return
			}
			c.stateChangedEventSources = append(c.stateChangedEventSources,
				c.addEventSource(eventSources),
			)
		}
	}
}

func (c *Character) addEventSource(eventSource events.EventSource) events.IDEventSource {
	return c.eventSourceMultiplexer.AddEventSource(eventSource)
}

func (c *Character) removeEventSource(id events.IDEventSource) {
	c.eventSourceMultiplexer.RemoveEventSource(id)
}
