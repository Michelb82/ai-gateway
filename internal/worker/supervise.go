package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Supervise repeatedly calls run until ctx is cancelled.
// On unexpected run errors or panics it logs and waits backoff before restarting.
func Supervise(ctx context.Context, run func(context.Context) error, logger *slog.Logger, backoff time.Duration) {
	if logger == nil {
		logger = slog.Default()
	}
	if backoff <= 0 {
		backoff = time.Second
	}

	for {
		if err := ctx.Err(); err != nil {
			return
		}

		err, panicked := runProtected(ctx, run)
		if ctx.Err() != nil {
			return
		}
		if panicked {
			logger.Error("worker panicked; restarting", "error", err, "backoff", backoff.String())
		} else if err != nil {
			logger.Error("worker stopped with error; restarting", "error", err, "backoff", backoff.String())
		} else {
			logger.Warn("worker exited unexpectedly; restarting", "backoff", backoff.String())
		}

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func runProtected(ctx context.Context, run func(context.Context) error) (err error, panicked bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			panicked = true
			err = panicAsError(recovered)
		}
	}()
	return run(ctx), false
}

func panicAsError(recovered any) error {
	if e, ok := recovered.(error); ok {
		return e
	}
	return fmt.Errorf("panic: %v", recovered)
}
