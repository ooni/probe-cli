package miniengine

//
// Log and progress events.
//

// EventTypeDebug is an [Event] containing a DEBUG message.
const EventTypeDebug = "DEBUG"

// EventTypeInfo is an [Event] containing an INFO message.
const EventTypeInfo = "INFO"

// EventTypeProgress is an [Event] containing a PROGRESS message.
const EventTypeProgress = "PROGRESS"

// EventTypeWarning is an [Event] containing a WARNING message.
const EventTypeWarning = "WARNING"

// Event is an interim event emitted by this implementation.
type Event struct {
	// EventType is one of "DEBUG", "INFO", "PROGRESS", and "WARNING".
	EventType string

	// Message is the string message.
	Message string

	// Progress is the progress as a number between zero and one.
	Progress float64
}
