package higgsfield_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AoziruCake/higgsfield-action/internal/higgsfield"
)

const testAPIKey = "key-id:secret"

func TestSubmitImage_success(t *testing.T) {
	t.Parallel()

	var gotAuth, gotPath string
	var gotBody higgsfield.ImageRequest

	srv := httptest.NewServer(nil)
	t.Cleanup(srv.Close)
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(higgsfield.SubmitResponse{
			Status:    higgsfield.StatusQueued,
			RequestID: "req-1",
			StatusURL: srv.URL + "/requests/req-1/status",
			CancelURL: srv.URL + "/requests/req-1/cancel",
		})
	})

	client := higgsfield.NewClient(testAPIKey, srv.Client()).WithBaseURL(srv.URL)
	resp, err := client.SubmitImage(context.Background(), "higgsfield-ai/soul/v2/standard", higgsfield.ImageRequest{
		Prompt:      "A cat",
		AspectRatio: "4:3",
		Resolution:  "720p",
	})
	if err != nil {
		t.Fatalf("SubmitImage: %v", err)
	}

	if gotAuth != "Key "+testAPIKey {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if !strings.HasSuffix(gotPath, "/higgsfield-ai/soul/v2/standard") {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody.Prompt != "A cat" || gotBody.AspectRatio != "4:3" {
		t.Fatalf("body = %+v", gotBody)
	}
	if resp.Status != higgsfield.StatusQueued || resp.RequestID != "req-1" {
		t.Fatalf("response = %+v", resp)
	}
	if resp.StatusURL == "" {
		t.Fatal("expected status_url")
	}
}

func TestSubmitImage_unauthorized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"detail": "Invalid credentials"})
	}))
	t.Cleanup(srv.Close)

	client := higgsfield.NewClient(testAPIKey, srv.Client()).WithBaseURL(srv.URL)
	_, err := client.SubmitImage(context.Background(), "higgsfield-ai/soul/v2/standard", higgsfield.ImageRequest{
		Prompt: "x",
	})
	var apiErr *higgsfield.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if !apiErr.IsUnauthorized() {
		t.Fatalf("expected unauthorized, got %+v", apiErr)
	}
}

func TestGetStatus_completed(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/requests/req-1/status" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(higgsfield.RequestStatus{
			Status:    higgsfield.StatusCompleted,
			RequestID: "req-1",
			Images:    []higgsfield.MediaOutput{{URL: "https://cdn.example/image.png"}},
		})
	}))
	t.Cleanup(srv.Close)

	client := higgsfield.NewClient(testAPIKey, srv.Client())
	st, err := client.GetStatus(context.Background(), srv.URL+"/requests/req-1/status")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if !st.Terminal() || st.Status != higgsfield.StatusCompleted {
		t.Fatalf("status = %+v", st)
	}
	url, ok := st.FirstImageURL()
	if !ok || url != "https://cdn.example/image.png" {
		t.Fatalf("FirstImageURL = %q, ok=%v", url, ok)
	}
}

func TestGetStatus_failed(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msg := "generation failed"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(higgsfield.RequestStatus{
			Status: higgsfield.StatusFailed,
			Error:  &msg,
		})
	}))
	t.Cleanup(srv.Close)

	client := higgsfield.NewClient(testAPIKey, srv.Client())
	st, err := client.GetStatus(context.Background(), srv.URL+"/status")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if st.Status != higgsfield.StatusFailed || !st.Terminal() {
		t.Fatalf("status = %+v", st)
	}
}

func TestRequestStatus_terminalStates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status   string
		terminal bool
	}{
		{higgsfield.StatusQueued, false},
		{higgsfield.StatusInProgress, false},
		{higgsfield.StatusCompleted, true},
		{higgsfield.StatusFailed, true},
		{higgsfield.StatusNSFW, true},
		{higgsfield.StatusCanceled, true},
	}

	for _, tc := range cases {
		st := higgsfield.RequestStatus{Status: tc.status}
		if st.Terminal() != tc.terminal {
			t.Fatalf("status %q: terminal=%v, want %v", tc.status, st.Terminal(), tc.terminal)
		}
	}
}
