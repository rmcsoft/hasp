package hasp

// Event is event Description
type Event struct {
	Name string
	Args []interface{}
}

// IDEventSource type to identify event sources
type IDEventSource = int

// EventSource is definition of the source of events.
type EventSource = chan *Event

// EventSources is set of event sources
type EventSources = []EventSource

// EventSourceMultiplexer implements work with event sources
type EventSourceMultiplexer struct {
	idSeq IDEventSource

	multiplexer  chan event
	eventSources map[IDEventSource]*eventSourceCtrl
}

// NewEventSourceMultiplexer creates new EventSourceMultiplexer
func NewEventSourceMultiplexer() *EventSourceMultiplexer {
	return &EventSourceMultiplexer{
		multiplexer:  make(chan event, 64),
		eventSources: make(map[IDEventSource]*eventSourceCtrl),
	}
}

// NextEvent gets next event
func (esm *EventSourceMultiplexer) NextEvent() *Event {
	for {
		e, ok := <-esm.multiplexer // Get next event
		if !ok {
			return nil
		}

		_, ok = esm.eventSources[e.idEventSource]
		if !ok { // The event is still relevant?
			continue
		}

		return e.event
	}
}

// AddEventSource adds new event source
func (esm *EventSourceMultiplexer) AddEventSource(src EventSource) IDEventSource {
	id := esm.idSeq
	esm.idSeq++

	newEventSourceCtrl := &eventSourceCtrl{
		src:  src,
		quit: make(chan bool),
	}

	esm.eventSources[id] = newEventSourceCtrl
	go esm.runEventSource(id, newEventSourceCtrl)

	return id
}

// RemoveEventSource removes event source
func (esm *EventSourceMultiplexer) RemoveEventSource(id IDEventSource) {
	if eventSource, ok := esm.eventSources[id]; ok {
		eventSource.quit <- true
		delete(esm.eventSources, id)
	}
}

type eventSourceCtrl struct {
	src      EventSource
	quit     chan bool
	finished bool
}

type event struct {
	idEventSource IDEventSource
	event         *Event
}

func (esm *EventSourceMultiplexer) runEventSource(id IDEventSource, ctrl *eventSourceCtrl) {
	for {
		select {
		case e := <-ctrl.src:
			esm.multiplexer <- event{id, e}
		case <-ctrl.quit:
			return
		}
	}
}
