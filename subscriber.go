package esl

import "strings"

// subscriber represents an ESL event subscriber.
type subscriber struct {
	Names map[string]struct{} // event names to handle and custom flag
	Send  chan<- Event        // send channel
}

// newSubscriber creates a new subscriber with the given names and send channel.
// If no event names are provided, all events are handled.
//
// If the send channel is nil, it panics.
func newSubscriber(send chan<- Event, events ...string) subscriber {
	if send == nil {
		//nolint:forbidigo // I don't want to return only this error
		panic("send channel cannot be nil")
	}

	if len(events) == 0 { // all events should be handled
		return subscriber{Names: nil, Send: send}
	}

	eventNames := make(map[string]struct{}, len(events))

	for _, name := range events {
		if name == "" || name == "*" || strings.EqualFold(name, "all") {
			return subscriber{Names: nil, Send: send} // all events
		}

		name, _ := strings.CutPrefix(name, "CUSTOM ")
		eventNames[name] = struct{}{}
	}

	return subscriber{Names: eventNames, Send: send}
}

// Handle sends the event to the subscriber's send channel if the event
// is handled by this subscriber.
//
// Returns true if the event was handled.
func (s subscriber) Handle(e Event) bool {
	if _, ok := s.Names[e.Name()]; ok || len(s.Names) == 0 {
		s.Send <- e

		return true
	}

	return false
}
