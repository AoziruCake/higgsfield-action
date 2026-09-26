package higgsfield

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

// Official polling guidance: start at 2s, grow to 10s, add a little jitter.
const (
	defaultInitialInterval = 2 * time.Second
	defaultMaxInterval     = 10 * time.Second
	defaultBackoffFactor   = 1.5
	defaultMaxJitter       = 500 * time.Millisecond
)

// PollOptions configures status polling backoff.
// Sleeper and Jitter are injectables so tests do not wait in real time.
type PollOptions struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	BackoffFactor   float64
	MaxJitter       time.Duration
	Sleeper         Sleeper
	Jitter          func() float64 // 0..1; scaled by MaxJitter
}

// Sleeper waits between poll attempts.
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

// DefaultPollOptions matches Higgsfield polling guidance.
func DefaultPollOptions() PollOptions {
	return PollOptions{
		InitialInterval: defaultInitialInterval,
		MaxInterval:     defaultMaxInterval,
		BackoffFactor:   defaultBackoffFactor,
		MaxJitter:       defaultMaxJitter,
		Sleeper:         realSleeper{},
		Jitter:          rand.Float64,
	}
}

func (o PollOptions) withDefaults() PollOptions {
	if o.InitialInterval <= 0 {
		o.InitialInterval = defaultInitialInterval
	}
	if o.MaxInterval <= 0 {
		o.MaxInterval = defaultMaxInterval
	}
	if o.BackoffFactor <= 1 {
		o.BackoffFactor = defaultBackoffFactor
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
			// 4xx such as 401/404 is a config or identity problem; 5xx may be transient.
			if shouldStopPolling(err) {
				return RequestStatus{}, err
			}
			delay, err = opts.backoff(ctx, delay)
			if err != nil {
				return RequestStatus{}, err
			}
			continue
		}

		if status.Terminal() {
			if status.Status == StatusCompleted {
				return status, nil
			}
			return status, &StatusError{Status: status}
		}

		delay, err = opts.backoff(ctx, delay)
		if err != nil {
			return RequestStatus{}, err
		}
	}
}

func (o PollOptions) backoff(ctx context.Context, delay time.Duration) (time.Duration, error) {
	if err := o.Sleeper.Sleep(ctx, o.sleepDuration(delay)); err != nil {
		return delay, err
	}
	return nextPollDelay(delay, o), nil
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
