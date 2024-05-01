package esl

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Conn errors.
var (
	ErrAccessDenied    = errors.New("access denied")
	ErrInvalidPassword = errors.New("invalid password")
	ErrTimeout         = errors.New("timeout")
)

// Conn represents an ESL connection.
type Conn struct {
	r          *bufio.Reader // response reader
	w          *bufio.Writer // command writer
	mu         sync.Mutex    // to protect the writer
	cmdTimeout time.Duration // command timeout
}

// NewConn returns a new authenticated ESL connection.
func NewConn(ctx context.Context, rw io.ReadWriter, password string, cmdTimeout time.Duration) (*Conn, error) {
	conn := &Conn{
		r:          bufio.NewReader(rw),
		w:          bufio.NewWriter(rw),
		mu:         sync.Mutex{},
		cmdTimeout: cmdTimeout,
	}

	// authenticate
	if err := conn.withTimeout(ctx, func() error {
		return conn.auth(password)
	}); err != nil {
		return nil, err
	}

	return conn, nil
}

// Write writes a command to the connection.
//
//nolint:errcheck // writing to the buffer never returns an error
func (c *Conn) Write(cmd string) error {
	if cmd == "" {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.w.WriteString(cmd)
	c.w.WriteString("\n\n")

	if err := c.w.Flush(); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	return nil
}

// Read reads the response from the connection.
//
// It reads the response line by line from the connection and
// parses the header values. It handles different header keys
// such as "Content-Type", "Reply-Text", "Job-UUID", and
// "Content-Length". If the "Content-Length" header is present,
// it reads the specified number of bytes as the response body.
// Finally, it logs the received response and returns it along
// with any error encountered during the process.
func (c *Conn) Read() (Response, error) {
	var (
		resp          Response
		contentLength int
	)

	for {
		line, err := c.readLine()
		if err != nil {
			return resp, err
		}

		if len(line) == 0 {
			if resp.ContentType == "" {
				continue // skip empty response
			}

			break // the end of response header
		}

		// parse response header
		idx := bytes.IndexByte(line, ':')
		if idx <= 0 {
			return resp, fmt.Errorf("malformed header line: %q", line)
		}

		key, value := string(line[:idx]), trimLeft(line[idx+1:])
		switch key {
		case "Content-Type":
			resp.ContentType = value
		case "Reply-Text":
			resp.Text = value
		case "Job-UUID":
			resp.JobUUID = value
		case "Content-Length":
			contentLength, err = strconv.Atoi(value)
			if err != nil {
				return resp, fmt.Errorf("malformed content-length: %q", value)
			}
		default: // ignore unsupported headers
		}
	}

	// read response body
	if contentLength > 0 {
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(c.r, body); err != nil {
			return resp, fmt.Errorf("failed to read response body: %w", err)
		}

		resp.Body = string(body)
	}

	return resp, nil
}

// Send sends a command to the connection and return Response.
// It's a shortcut for c.Write and c.Read.
func (c *Conn) Send(cmd string) (Response, error) {
	if err := c.Write(cmd); err != nil {
		return Response{}, fmt.Errorf("send: %w", err)
	}

	resp, err := c.Read()
	if err != nil {
		return Response{}, fmt.Errorf("read: %w", err)
	}

	return resp, nil
}

// SendCtx sends a command to the connection and return Response with context and command timeout.
func (c *Conn) SendCtx(ctx context.Context, cmd string) (Response, error) {
	var resp Response

	if err := c.withTimeout(ctx, func() error {
		var err error
		resp, err = c.Send(cmd)

		return err
	}); err != nil {
		return Response{}, err
	}

	return resp, nil
}

// withTimeout executes the given function with a timeout context.
//
// If the timeout is reached, it returns ErrTimeout.
// If the context is canceled, it returns context.Cause(ctx).
// If the function returns an error, it returns the error.
// Otherwise, it returns nil.
func (c *Conn) withTimeout(ctx context.Context, f func() error) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- f()
		close(errCh)
	}()

	if c.cmdTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeoutCause(ctx, c.cmdTimeout, ErrTimeout)
		defer cancel()
	}

	select {
	case <-ctx.Done():
		//nolint:wrapcheck // return the original context error
		return context.Cause(ctx)
	case err := <-errCh:
		return err
	}
}

// auth authenticates the connection using the provided password.
//
// It reads the server response, validates the content type, and sends the authentication request.
// Returns an error if the request fails or the response is unexpected.
func (c *Conn) auth(password string) error {
	resp, err := c.Read()
	if err != nil {
		return fmt.Errorf("read server request: %w", err)
	}

	switch resp.ContentType {
	default:
		return fmt.Errorf("unexpected auth request content type: %s", resp.ContentType)
	case ctReject:
		return ErrAccessDenied
	case ctDisconnect:
		return fmt.Errorf("server disconnect: %w", io.EOF)
	case ctAuth: // OK
	}

	switch resp, err := c.Send("auth " + password); {
	case err != nil:
		return fmt.Errorf("read auth response: %w", err)
	case resp.ContentType != ctCommandReply:
		return fmt.Errorf("unexpected auth response content type: %s", resp.ContentType)
	case !strings.HasPrefix(resp.Text, "+OK"):
		return ErrInvalidPassword
	default:
		return nil // OK
	}
}

// readLine reads a line from the conn's reader.
func (c *Conn) readLine() ([]byte, error) {
	var fullLine []byte // to accumulate full line

	for {
		line, more, err := c.r.ReadLine()
		if err != nil {
			return nil, err //nolint:wrapcheck
		}

		if fullLine == nil && !more {
			return line, nil // the whole line is read at once
		}

		fullLine = append(fullLine, line...) // accumulate

		if !more {
			return fullLine, nil // it's the end of line
		}
	}
}

// trimLeft removes leading spaces and tabs from the given byte slice and returns the result as a string.
func trimLeft(b []byte) string {
	for i := range len(b) {
		if b[i] != ' ' && b[i] != '\t' {
			return string(b[i:])
		}
	}

	return ""
}
