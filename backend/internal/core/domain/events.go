package domain

import "time"

// EventType categorises an event for routing to the right handlers.
type EventType string

const (
	EventEmailSent   EventType = "email.sent"
	EventEmailFailed EventType = "email.failed"
)

// Event represents something that happened in the system.
// Handlers subscribe by EventType and receive a copy of the Event.
type Event struct {
	Type EventType
	Data any
	Time time.Time
}

// NewEvent builds an Event with the current timestamp.
func NewEvent(typ EventType, data any) Event {
	return Event{Type: typ, Data: data, Time: time.Now()}
}
