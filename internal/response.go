package esl

import (
	"errors"
	"io"
	"log/slog"
	"strings"
)

// Response content types.
const (
	ctCommandReply = "command/reply"
	ctAPIResponse  = "api/response"
	ctAuth         = "auth/request"
	ctEventPlain   = "text/event-plain"
	ctEventJSON    = "text/event-json"
	ctEventXML     = "text/event-xml"
	ctDisconnect   = "text/disconnect-notice"
	ctReject       = "text/rude-rejection"
)

// Response represents an ESL Response with headers and a body.
type Response struct {
	ContentType string // Content-Type
	Text        string // Reply-Text
	JobUUID     string // Job-UUID
	Body        string // Body
}

// AsErr checks the content type of the response and returns an error if it matches a specific case.
func (r Response) AsErr() error {
	switch r.ContentType {
	case ctDisconnect:
		return io.EOF
	case ctCommandReply:
		return responseError(r.Text)
	case ctAPIResponse:
		return responseError(r.Body)
	default:
		return nil
	}
}

// responseError parses the given string as an error and returns it if text starts with "-ERR " prefix.
func responseError(text string) error {
	const errPrefix = "-ERR "
	if text, ok := strings.CutPrefix(text, errPrefix); ok {
		return errors.New(text)
	}

	return nil
}

// LogValue returns a slog.Value object that represents the log attributes for the response.
func (r Response) LogValue() slog.Value {
	attr := make([]slog.Attr, 0, 3)
	attr = append(attr, slog.String("type", r.ContentType))

	if r.JobUUID != "" {
		attr = append(attr, slog.String("job-uuid", r.JobUUID))
	}

	if err := r.AsErr(); err != nil {
		attr = append(attr, slog.String("error", err.Error()))
	} else if length := len(r.Body); length > 0 {
		attr = append(attr, slog.Int("length", length))
	}

	return slog.GroupValue(attr...)
}
