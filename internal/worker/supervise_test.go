package worker_test

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mywebsite/construction-ai-gateway/internal/worker"
)

func TestSuperviseRestartsAfterErrorThenStopsOnCancel(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Supervise(ctx, func(runCtx context.Context) error {
			n := calls.Add(1)
			if n == 1 {
				return errors.New("transient failure")
			}
			<-runCtx.Done()
			return runCtx.Err()
		}, slog.Default(), 5*time.Millisecond)
	}()

	waitCalls(t, &calls, 2)
	cancel()
	waitDone(t, done)

	if got := calls.Load(); got < 2 {
		t.Fatalf("calls = %d, want at least 2", got)
	}
}

func TestSuperviseRestartsAfterNilExit(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Supervise(ctx, func(runCtx context.Context) error {
			n := calls.Add(1)
			if n == 1 {
				return nil
			}
			<-runCtx.Done()
			return runCtx.Err()
		}, slog.Default(), 5*time.Millisecond)
	}()

	waitCalls(t, &calls, 2)
	cancel()
	waitDone(t, done)
}

func TestSuperviseRestartsAfterPanic(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Supervise(ctx, func(runCtx context.Context) error {
			n := calls.Add(1)
			if n == 1 {
				panic("boom")
			}
			<-runCtx.Done()
			return runCtx.Err()
		}, slog.Default(), 5*time.Millisecond)
	}()

	waitCalls(t, &calls, 2)
	cancel()
	waitDone(t, done)
}

func TestSuperviseExitsDuringBackoffOnCancel(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Supervise(ctx, func(context.Context) error {
			calls.Add(1)
			return errors.New("fail")
		}, slog.Default(), time.Hour)
	}()

	waitCalls(t, &calls, 1)
	cancel()
	waitDone(t, done)

	if got := calls.Load(); got != 1 {
		t.Fatalf("calls = %d, want 1 (no restart during cancelled backoff)", got)
	}
}

func waitCalls(t *testing.T, calls *atomic.Int32, want int32) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for calls.Load() < want {
		select {
		case <-deadline:
			t.Fatalf("Supervise calls = %d, want at least %d", calls.Load(), want)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func waitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Supervise did not exit after cancel")
	}
}
