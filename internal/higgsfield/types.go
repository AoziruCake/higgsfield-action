package higgsfield

import (
	"fmt"
	"strings"
)

// Status is a Higgsfield request lifecycle value from the official API.
type Status string

const (
	StatusQueued     Status = "queued"      // waiting to start; may still be canceled
	StatusInProgress Status = "in_progress" // generation has started
	StatusCompleted  Status = "completed"   // images (or other media) are available
	StatusFailed     Status = "failed"      // generation failed; Error may explain why
	StatusNSFW       Status = "nsfw"        // rejected by content moderation
	StatusCanceled   Status = "canceled"    // canceled before processing started
)

// Terminal reports whether this status is final (stop polling).
func (s Status) Terminal() bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusNSFW, StatusCanceled:
		return true
	default:
		return false
	}
}

// FailureMessage is the user-facing text for a non-success terminal status.
func (s Status) FailureMessage(detail *string) string {
	switch s {
	case StatusFailed:
		if detail != nil && *detail != "" {
			return fmt.Sprintf("generation failed: %s", *detail)
		}
		return "generation failed"
	case StatusNSFW:
		return "generation rejected by content moderation"
	case StatusCanceled:
		return "generation was canceled"
	default:
		return fmt.Sprintf("generation ended with status %q", s)
	}
}

// ImageRequest is the JSON body for Soul-style text-to-image endpoints.
type ImageRequest struct {
	Prompt      string `json:"prompt"`
	AspectRatio string `json:"aspect_ratio"`
	Resolution  string `json:"resolution"`
}

// SubmitResponse is returned immediately after a successful submission.
// Poll using StatusURL as given; do not reconstruct it from RequestID.
type SubmitResponse struct {
	Status    Status `json:"status"`
	RequestID string `json:"request_id"`
	StatusURL string `json:"status_url"`
	CancelURL string `json:"cancel_url"`
}

// Validate checks that polling can start from this submission.
func (s SubmitResponse) Validate() error {
	if strings.TrimSpace(s.RequestID) == "" {
		return fmt.Errorf("submit response missing request_id")
	}
	if strings.TrimSpace(s.StatusURL) == "" {
		return fmt.Errorf("submit response missing status_url")
	}
	return nil
}

// MediaOutput holds a downloadable artifact URL.
type MediaOutput struct {
	URL string `json:"url"`
}

// RequestStatus is the current state of a generation request.
type RequestStatus struct {
	Status    Status        `json:"status"`
	RequestID string        `json:"request_id"`
	StatusURL string        `json:"status_url,omitempty"`
	CancelURL string        `json:"cancel_url,omitempty"`
	Error     *string       `json:"error"`
	Images    []MediaOutput `json:"images,omitempty"`
	Video     *MediaOutput  `json:"video,omitempty"` // reserved for later video models
}

// Terminal reports whether the request reached a final state.
func (s RequestStatus) Terminal() bool {
	return s.Status.Terminal()
}

// FailureMessage is the user-facing text for a failed terminal request.
func (s RequestStatus) FailureMessage() string {
	return s.Status.FailureMessage(s.Error)
}

// FirstImageURL returns the first image URL, if any.
func (s RequestStatus) FirstImageURL() (string, bool) {
	if len(s.Images) == 0 || s.Images[0].URL == "" {
		return "", false
	}
	return s.Images[0].URL, true
}
