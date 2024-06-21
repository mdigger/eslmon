package esl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"strings"
	"time"

	esl "github.com/mdigger/eslmon/internal"
)

// Monitor errors.
var (
	ErrNotConnected    = errors.New("not connected")
	ErrAccessDenied    = errors.New("access denied")
	ErrInvalidPassword = errors.New("invalid password")
	ErrTimeout         = errors.New("timeout")
)

// Monitor represents a FreeSWITCH ESL Monitor instance.
type Monitor struct {
	addr, password string
	dialer         *net.Dialer
	subscribers    []subscriber
	cmdTimeout     time.Duration
}

// New creates a new FreeSWITCH ESL Monitor instance.
//
// If the address doesn't contain a port, use the default port (8021).
// Panic if the address is malformed.
func New(addr, password string) *Monitor {
	const (
		dialTimeout         = time.Second * 5 // dialer timeout
		cmdTimeout          = time.Second * 5 // command timeout
		subscribersCapacity = 10              // capacity for the subscribers slice
	)

	return &Monitor{
		addr:        addAddrPort(addr),
		password:    password,
		dialer:      &net.Dialer{Timeout: dialTimeout}, //nolint:exhaustruct
		subscribers: make([]subscriber, 0, subscribersCapacity),
		cmdTimeout:  cmdTimeout,
	}
}

// Subscribe adds a new subscriber to the Monitor.
//
// The send channel is used to send events to the subscriber.
//
// The events parameter is a list of event names.
// If no events are provided or the "*" wildcard is used, all events are subscribed.
func (m *Monitor) Subscribe(send chan<- Event, events ...string) *Monitor {
	m.subscribers = append(m.subscribers, newSubscriber(send, events...))

	return m
}

// Run connects to the ESL server and subscribes to the events.
//
// The connection is closed when the context is canceled or expired, and an error is returned.
// The error is the context error.
//
// Returns an error if the connection fails or the authentication fails.
func (m *Monitor) Run(ctx context.Context) error {
	conn, err := m.dialer.DialContext(ctx, "tcp", m.addr)
	if err != nil {
		return fmt.Errorf("dialer: %w", err)
	}

	// disconnect after the context is done or exit with error
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	context.AfterFunc(ctx, func() { conn.Close() })

	// init ESL connection and authenticate
	eslConn, err := esl.NewConn(ctx, conn, m.password, m.cmdTimeout)
	if err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}

	// subscribe to the ESL events if subscribers are set
	if len(m.subscribers) > 0 {
		resp, err := eslConn.SendCtx(ctx, m.subscribe())
		if err != nil {
			return fmt.Errorf("subscribe: %w", err)
		}

		if err = resp.AsErr(); err != nil {
			return fmt.Errorf("subscribe response: %w", err)
		}
	}

	for {
		resp, err := eslConn.Read()
		if err != nil {
			if err := context.Cause(ctx); err != nil {
				return fmt.Errorf("done: %w", err) // context error
			}

			return fmt.Errorf("read: %w", err) // read error
		}

		switch resp.ContentType {
		case "text/event-plain":
			event, err := parseEvent(resp.Body)
			if err != nil {
				return fmt.Errorf("event parse: %w", err)
			}

			for _, subscriber := range m.subscribers {
				subscriber.Handle(event)
			}

		case "text/disconnect-notice":
			return fmt.Errorf("server closed: %w", io.EOF)
		}
	}
}

// WithDialTimeout sets the dialer timeout.
func (m *Monitor) WithDialTimeout(timeout time.Duration) *Monitor {
	m.dialer.Timeout = timeout

	return m
}

// WithCommandsTimeout sets the command timeout.
// The default command timeout is 5 seconds.
//
// Used on authorization and subscription requests.
func (m *Monitor) WithCommandsTimeout(timeout time.Duration) *Monitor {
	m.cmdTimeout = timeout

	return m
}

// subscribe returns the command string with ESL event names to subscribe.
func (m *Monitor) subscribe() string {
	const (
		cmdSubscribe   = "event plain"
		eventsCapacity = 100
	)

	events := make(map[string]struct{}, eventsCapacity)

	for _, subscriber := range m.subscribers {
		if len(subscriber.Names) == 0 {
			return cmdSubscribe + " ALL" // all events should be handled
		}

		maps.Copy(events, subscriber.Names)
	}

	var cmd, custom strings.Builder

	cmd.WriteString(cmdSubscribe)

	for name := range events {
		if _, ok := eventNames[name]; ok {
			cmd.WriteByte(' ')
			cmd.WriteString(name)
		} else {
			custom.WriteByte(' ')
			custom.WriteString(name)
		}
	}

	if custom.Len() > 0 {
		if cmd.Len() > 0 {
			cmd.WriteByte(' ')
		}

		cmd.WriteString("CUSTOM")
		cmd.WriteString(custom.String())
	}

	return cmd.String()
}

// addAddrPort adds a default port to the given address if it doesn't contain a port.
// If the address contains a port, it is returned as is.
// Panics if the address is invalid.
func addAddrPort(addr string) string {
	// if the address doesn't contain a port, use the default port
	if _, _, err := net.SplitHostPort(addr); err != nil {
		var addrErr *net.AddrError
		if !errors.As(err, &addrErr) || addrErr.Err != "missing port in address" {
			//nolint:forbidigo // I don't want to return an error only for this
			panic(fmt.Errorf("bad address: %w", err))
		}

		const defaultPort = "8021"
		addr = net.JoinHostPort(addr, defaultPort)
	}

	return addr
}
