package esl

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// Event represents an ESL event with headers and a body.
type Event map[string]string

// Get returns the value associated with the given key from the Event's headers.
func (e Event) Get(key string) string {
	return e[key]
}

// Event constants.
const (
	contentLengthKey  = "Content-Length"
	eventNameKey      = "Event-Name"
	eventSubclassKey  = "Event-Subclass"
	eventSequenceKey  = "Event-Sequence"
	eventTimestampKey = "Event-Date-Timestamp"
	eventJobUUIDKey   = "Job-UUID"
	variableKeyPrefix = "variable_"
	bodyKey           = "_body"
)

// Name returns the name of the event.
func (e Event) Name() string {
	if name := e.Get(eventSubclassKey); name != "" {
		return name
	}

	return e.Get(eventNameKey)
}

// ContentType returns the content type of the event.
func (e Event) ContentType() string {
	return e.Get(contentLengthKey)
}

// Body returns the body of the event as a string.
func (e Event) Body() string {
	return e.Get(bodyKey)
}

// ContentLength returns the length of the body in the Event.
func (e Event) ContentLength() int {
	return len(e.Body())
}

// Sequence returns the event sequence as an int64.
func (e Event) Sequence() int64 {
	i, _ := strconv.ParseInt(e.Get(eventSequenceKey), 10, 64)

	return i
}

// Timestamp returns the timestamp of the event.
func (e Event) Timestamp() time.Time {
	ts := e.Get(eventTimestampKey)
	if i, err := strconv.ParseInt(ts, 10, 64); err == nil {
		return time.UnixMicro(i)
	}

	return time.Time{}
}

// Variable returns the value of the variable with the given name.
func (e Event) Variable(name string) string {
	return e.Get(variableKeyPrefix + name)
}

// IsCustom returns true if the event is a custom event.
func (e Event) IsCustom() bool {
	return e[eventNameKey] == "CUSTOM"
}

// LogValue returns the log value of the Event.
//
// It returns a slog.Value that contains the name and sequence of the Event.
func (e Event) LogValue() slog.Value {
	attr := make([]slog.Attr, 0, 3)
	attr = append(attr,
		slog.String("name", e.Name()),
		slog.Int64("sequence", e.Sequence()),
	)

	if jobUUID := e.Get(eventJobUUIDKey); jobUUID != "" {
		attr = append(attr, slog.String("job-uuid", jobUUID))
	}

	return slog.GroupValue(attr...)
}

// parseEvent parses the given body as an ESL event in JSON format and returns it.
func parseEvent(body string) (Event, error) {
	var event Event
	if err := json.Unmarshal([]byte(body), &event); err != nil {
		return nil, fmt.Errorf("parse event: %w", err)
	}

	return event, nil
}

// eventNames is a map that contains the predefined names of various events as keys.
var eventNames = map[string]struct{}{ //nolint:gochecknoglobals
	// spell-checker:disable
	// "CUSTOM":                   {},
	"CLONE":                    {},
	"CHANNEL_CREATE":           {},
	"CHANNEL_DESTROY":          {},
	"CHANNEL_STATE":            {},
	"CHANNEL_CALLSTATE":        {},
	"CHANNEL_ANSWER":           {},
	"CHANNEL_HANGUP":           {},
	"CHANNEL_HANGUP_COMPLETE":  {},
	"CHANNEL_EXECUTE":          {},
	"CHANNEL_EXECUTE_COMPLETE": {},
	"CHANNEL_HOLD":             {},
	"CHANNEL_UNHOLD":           {},
	"CHANNEL_BRIDGE":           {},
	"CHANNEL_UNBRIDGE":         {},
	"CHANNEL_PROGRESS":         {},
	"CHANNEL_PROGRESS_MEDIA":   {},
	"CHANNEL_OUTGOING":         {},
	"CHANNEL_PARK":             {},
	"CHANNEL_UNPARK":           {},
	"CHANNEL_APPLICATION":      {},
	"CHANNEL_ORIGINATE":        {},
	"CHANNEL_UUID":             {},
	"API":                      {},
	"LOG":                      {},
	"INBOUND_CHAN":             {},
	"OUTBOUND_CHAN":            {},
	"STARTUP":                  {},
	"SHUTDOWN":                 {},
	"PUBLISH":                  {},
	"UNPUBLISH":                {},
	"TALK":                     {},
	"NOTALK":                   {},
	"SESSION_CRASH":            {},
	"MODULE_LOAD":              {},
	"MODULE_UNLOAD":            {},
	"DTMF":                     {},
	"MESSAGE":                  {},
	"PRESENCE_IN":              {},
	"NOTIFY_IN":                {},
	"PRESENCE_OUT":             {},
	"PRESENCE_PROBE":           {},
	"MESSAGE_WAITING":          {},
	"MESSAGE_QUERY":            {},
	"ROSTER":                   {},
	"CODEC":                    {},
	"BACKGROUND_JOB":           {},
	"DETECTED_SPEECH":          {},
	"DETECTED_TONE":            {},
	"PRIVATE_COMMAND":          {},
	"HEARTBEAT":                {},
	"TRAP":                     {},
	"ADD_SCHEDULE":             {},
	"DEL_SCHEDULE":             {},
	"EXE_SCHEDULE":             {},
	"RE_SCHEDULE":              {},
	"RELOADXML":                {},
	"NOTIFY":                   {},
	"PHONE_FEATURE":            {},
	"PHONE_FEATURE_SUBSCRIBE":  {},
	"SEND_MESSAGE":             {},
	"RECV_MESSAGE":             {},
	"REQUEST_PARAMS":           {},
	"CHANNEL_DATA":             {},
	"GENERAL":                  {},
	"COMMAND":                  {},
	"SESSION_HEARTBEAT":        {},
	"CLIENT_DISCONNECTED":      {},
	"SERVER_DISCONNECTED":      {},
	"SEND_INFO":                {},
	"RECV_INFO":                {},
	"RECV_RTCP_MESSAGE":        {},
	"SEND_RTCP_MESSAGE":        {},
	"CALL_SECURE":              {},
	"NAT":                      {},
	"RECORD_START":             {},
	"RECORD_STOP":              {},
	"PLAYBACK_START":           {},
	"PLAYBACK_STOP":            {},
	"CALL_UPDATE":              {},
	"FAILURE":                  {},
	"SOCKET_DATA":              {},
	"MEDIA_BUG_START":          {},
	"MEDIA_BUG_STOP":           {},
	"CONFERENCE_DATA_QUERY":    {},
	"CONFERENCE_DATA":          {},
	"CALL_SETUP_REQ":           {},
	"CALL_SETUP_RESULT":        {},
	"CALL_DETAIL":              {},
	"DEVICE_STATE":             {},
	"TEXT":                     {},
	"SHUTDOWN_REQUESTED":       {},
}
