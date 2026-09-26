package higgsfield

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

// PollOptions configures status polling backoff.
type PollOptions struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	BackoffFactor   float64
	MaxJitter       time.Duration
	Sleeper         Sleeper
	Jitter          func() float64
}

// Sleeper waits between poll attempts (injectable for tests).
type Sleeper interface {
	Sleep(context.Context, time.Duration) error
}

type realSleeper struct{}

func (realSleeper) Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// DefaultPollOptions matches Higgsfield polling guidance (2s start, up to 10s, jitter).
func DefaultPollOptions() PollOptions {
	return PollOptions{
		InitialInterval: 2 * time.Second,
		MaxInterval:     10 * time.Second,
		BackoffFactor:   1.5,
		MaxJitter:       500 * time.Millisecond,
		Sleeper:         realSleeper{},
		Jitter:          rand.Float64,
	}
}

func (o PollOptions) withDefaults() PollOptions {
	if o.InitialInterval <= 0 {
		o.InitialInterval = 2 * time.Second
	}
	if o.MaxInterval <= 0 {
		o.MaxInterval = 10 * time.Second
	}
	if o.BackoffFactor <= 1 {
		o.BackoffFactor = 1.5
	}
	if o.MaxJitter < 0 {
		o.MaxJitter = 0
	}
	if o.Sleeper == nil {
		o.Sleeper = realSleeper{}
	}
	if o.Jitter == nil {
		o.Jitter = rand.Float64
	}
	return o
}

// WaitForCompletion polls statusURL until a terminal state or ctx is canceled.
func (c *Client) WaitForCompletion(ctx context.Context, statusURL string, opts PollOptions) (RequestStatus, error) {
	opts = opts.withDefaults()
	delay := opts.InitialInterval

	for {
		if err := ctx.Err(); err != nil {
			return RequestStatus{}, err
		}

		status, err := c.GetStatus(ctx, statusURL)
		if err != nil {
			if shouldStopPolling(err) {
				return RequestStatus{}, err
			}
			if err := opts.Sleeper.Sleep(ctx, opts.sleepDuration(delay)); err != nil {
				return RequestStatus{}, err
			}
			delay = nextPollDelay(delay, opts)
			continue
		}

		if status.Terminal() {
			if status.Status == StatusCompleted {
				return status, nil
			}
			return status, &StatusError{Status: status}
		}

		if err := opts.Sleeper.Sleep(ctx, opts.sleepDuration(delay)); err != nil {
			return RequestStatus{}, err
		}
		delay = nextPollDelay(delay, opts)
	}
}

func shouldStopPolling(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.IsUnauthorized() || apiErr.IsNotFound() {
			return true
		}
		if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			return true
		}
	}
	return false
}

func (o PollOptions) sleepDuration(base time.Duration) time.Duration {
	if o.MaxJitter == 0 {
		return base
	}
	return base + time.Duration(o.Jitter()*float64(o.MaxJitter))
}

func nextPollDelay(current time.Duration, opts PollOptions) time.Duration {
	next := time.Duration(float64(current) * opts.BackoffFactor)
	if next > opts.MaxInterval {
		return opts.MaxInterval
	}
	return next
}
