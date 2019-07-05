package hasp

type singleEventSource struct {
	name      string
	eventChan chan *Event
}

// NewSingleEventSource creates new singleEventSource
func NewSingleEventSource(name string, fn func() *Event) EventSource {
	es := &singleEventSource{
		name:      name,
		eventChan: make(chan *Event),
	}

	go func() {
		es.eventChan <- fn()
		close(es.eventChan)
	}()

	return es
}

func (es *singleEventSource) Name() string {
	return es.name
}

func (es *singleEventSource) Events() chan *Event {
	return es.eventChan
}

func (es *singleEventSource) Close() {
}
