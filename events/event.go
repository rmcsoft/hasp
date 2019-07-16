package events

import (
	log "github.com/sirupsen/logrus"
)

// Event is event Description
type Event struct {
	Name string
	Args []interface{}
}

// StateChangedEventName An event means that the current state has changed and
// need to update the animation and sound without naming the current state
const StateChangedEventName = "StateChanged"
const StateGoIdleName = "GoIdle"

// IDEventSource type to identify event sources
type IDEventSource = uint64

// EventSource is definition of the source of events.
type EventSource interface {
	Name() string
	Events() chan *Event
	Close()
}

// EventSources is set of event sources
type EventSources = []EventSource

// EventSourceMultiplexer implements work with event sources
type EventSourceMultiplexer struct {
	idSeq IDEventSource

	multiplexer  chan event
	eventSources map[IDEventSource]EventSource
}

// NewEventSourceMultiplexer creates new EventSourceMultiplexer
func NewEventSourceMultiplexer() *EventSourceMultiplexer {
	return &EventSourceMultiplexer{
		multiplexer:  make(chan event, 64),
		eventSources: make(map[IDEventSource]EventSource),
	}
}

// NextEvent gets next event
func (esm *EventSourceMultiplexer) NextEvent() *Event {
	for {
		e, ok := <-esm.multiplexer // Get next event
		if !ok {
			return nil
		}

		es, ok := esm.eventSources[e.idEventSource]
		if !ok { // The event is still relevant?
			continue
		}

		log.Infof("NextEvent: Source=%s, Name=%s\n",
			es.Name(), e.event.Name)
		return e.event
	}
}

// AddEventSource adds new event source
func (esm *EventSourceMultiplexer) AddEventSource(eventSource EventSource) IDEventSource {
	id := esm.idSeq
	esm.idSeq++

	esm.eventSources[id] = eventSource
	go esm.runEventSource(id, eventSource)

	return id
}

// RemoveEventSource removes event source
func (esm *EventSourceMultiplexer) RemoveEventSource(id IDEventSource) {
	if eventSource, ok := esm.eventSources[id]; ok {
		eventSource.Close()
		delete(esm.eventSources, id)
	}
}

type eventSourceCtrl struct {
	src  EventSource
	quit chan bool
}

type event struct {
	idEventSource IDEventSource
	event         *Event
}

func (esm *EventSourceMultiplexer) runEventSource(id IDEventSource, eventSource EventSource) {
	log.Infof("EventSource '%s' running\n", eventSource.Name())
	for e := range eventSource.Events() {
		esm.multiplexer <- event{id, e}
	}
	log.Infof("EventSource '%s' stopped\n", eventSource.Name())
}
