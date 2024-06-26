package esl

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestRealConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	eventsChan := make(chan Event)
	go func() {
		for e := range eventsChan {
			t.Logf("event: %s", e.Name())
		}
	}()

	const timeout = time.Second * 20

	ctx, cancel := context.WithTimeoutCause(
		context.Background(), timeout, errors.New("the end"))
	defer cancel()

	monitor := New(
		os.Getenv("ESL_ADDR"),
		os.Getenv("ESL_PASSWORD"))
	monitor.
		// Subscribe(eventsChan).
		Subscribe(eventsChan, "HEARTBEAT", "RE_SCHEDULE")

	t.Error(monitor.Run(ctx))
}
