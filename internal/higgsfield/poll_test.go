package higgsfield_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AoziruCake/higgsfield-action/internal/higgsfield"
)

func instantPollOptions() higgsfield.PollOptions {
	opts := higgsfield.DefaultPollOptions()
	opts.Sleeper = sleeperFunc(func(context.Context, time.Duration) error { return nil })
	opts.Jitter = func() float64 { return 0 }
	return opts
}

type sleeperFunc func(context.Context, time.Duration) error

func (f sleeperFunc) Sleep(ctx context.Context, d time.Duration) error {
	return f(ctx, d)
}

func TestWaitForCompletion_eventuallyCompleted(t *testing.T) {
	t.Parallel()

	var polls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := polls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n < 3 {
			_ = json.NewEncoder(w).Encode(higgsfield.RequestStatus{Status: higgsfield.StatusInProgress})
			return
		}
		_ = json.NewEncoder(w).Encode(higgsfield.RequestStatus{
			Status:    higgsfield.StatusCompleted,
			RequestID: "req-1",
			Images:    []higgsfield.MediaOutput{{URL: "https://cdn.example/a.png"}},
		})
	}))
	t.Cleanup(srv.Close)

	client := higgsfield.NewClient(testAPIKey, srv.Client())
	st, err := client.WaitForCompletion(context.Background(), srv.URL, instantPollOptions())
	if err != nil {
		t.Fatalf("WaitForCompletion: %v", err)
	}
	if polls.Load() < 3 {
		t.Fatalf("polls = %d, want >= 3", polls.Load())
	}
	if st.Status != higgsfield.StatusCompleted {
		t.Fatalf("status = %+v", st)
	}
}

func TestWaitForCompletion_failedTerminal(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msg := "model error"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(higgsfield.RequestStatus{
			Status: higgsfield.StatusFailed,
			Error:  &msg,
		})
	}))
	t.Cleanup(srv.Close)

	client := higgsfield.NewClient(testAPIKey, srv.Client())
	_, err := client.WaitForCompletion(context.Background(), srv.URL, instantPollOptions())
	var statusErr *higgsfield.StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("expected StatusError, got %v", err)
	}
}

func TestWaitForCompletion_unauthorizedStops(t *testing.T) {
	t.Parallel()

	var polls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		polls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"detail": "Invalid credentials"})
	}))
	t.Cleanup(srv.Close)

	client := higgsfield.NewClient(testAPIKey, srv.Client())
	_, err := client.WaitForCompletion(context.Background(), srv.URL, instantPollOptions())
	var apiErr *higgsfield.APIError
	if !errors.As(err, &apiErr) || !apiErr.IsUnauthorized() {
		t.Fatalf("expected unauthorized APIError, got %v", err)
	}
	if polls.Load() != 1 {
		t.Fatalf("polls = %d, want 1 (no retry on 401)", polls.Load())
	}
}

func TestWaitForCompletion_retriesServerError(t *testing.T) {
	t.Parallel()

	var polls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := polls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(higgsfield.RequestStatus{Status: higgsfield.StatusCompleted})
	}))
	t.Cleanup(srv.Close)

	client := higgsfield.NewClient(testAPIKey, srv.Client())
	_, err := client.WaitForCompletion(context.Background(), srv.URL, instantPollOptions())
	if err != nil {
		t.Fatalf("WaitForCompletion: %v", err)
	}
	if polls.Load() != 2 {
		t.Fatalf("polls = %d, want 2", polls.Load())
	}
}

func TestWaitForCompletion_respectsContext(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(higgsfield.RequestStatus{Status: higgsfield.StatusQueued})
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	opts := higgsfield.DefaultPollOptions()
	client := higgsfield.NewClient(testAPIKey, srv.Client())
	_, err := client.WaitForCompletion(ctx, srv.URL, opts)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}
