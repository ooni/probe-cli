package torcontrol

// EventReader reads async events emitted by the [*Conn].
//
// Use Attach to attach to the related [*Conn]. Make sure you eventually
// call Detach to stop being attached to the related [*Conn].
//
// The zero value is invalid; use [NewEventReader] to instantiate.
//
// We internally use a buffered channel to avoid losing events and the
// [*Conn] read loop sends in nonblocking mode. This means that you must
// ensure to drain the channel to avoid missing events.
type EventReader struct {
	// conn is the owning conn.
	conn *Conn

	// events is the channel reading events.
	events chan *Response
}

// NewEventReader creates an [*EventReader] using the given [*Conn].
func NewEventReader(conn *Conn) *EventReader {
	const buffer = 128
	return &EventReader{
		conn:   conn,
		events: make(chan *Response, buffer),
	}
}

// Attach attaches an event reader to the corresponding [*Conn].
func (er *EventReader) Attach() {
	er.conn.attachReader(er)
}

// attachReader attaches an [*EventReader] to this [*Conn].
func (c *Conn) attachReader(er *EventReader) {
	// lock for writing
	defer c.eventReadersRWLock.Unlock()
	c.eventReadersRWLock.Lock()

	// replace the whole list
	var list []*EventReader
	list = append(list, c.eventReadersList...)
	list = append(list, er)
	c.eventReadersList = list
}

// Detach detaches an event reader from the corresponding [*Conn].
func (er *EventReader) Detach() {
	er.conn.detachReader(er)
}

// detachReader detaches an [*EventReader] from this [*Conn].
func (c *Conn) detachReader(er *EventReader) {
	// lock for writing
	defer c.eventReadersRWLock.Unlock()
	c.eventReadersRWLock.Lock()

	// replace the whole list
	var list []*EventReader
	for _, entry := range c.eventReadersList {
		if er != entry {
			list = append(list, entry)
		}
	}
	c.eventReadersList = list
}

// dispatchEvent dispatches an event [*Reponse] to all the [*EventReader]
// instances that are currently attached to this [*Conn].
func (c *Conn) dispatchEvent(ev *Response) {
	// lock for reading
	c.eventReadersRWLock.RLock()

	// build a list of channels to dispatch to
	var outputs []chan *Response
	for _, entry := range c.eventReadersList {
		outputs = append(outputs, entry.events)
	}

	// unlock for reading
	c.eventReadersRWLock.RUnlock()

	// dispatch event in nonblocking mode
	for _, output := range outputs {
		select {
		case output <- ev:
			// emitted
		default:
			// not emitted
		}
	}
}

// Events returns the channel from which one can read events.
func (er *EventReader) Events() <-chan *Response {
	return er.events
}
