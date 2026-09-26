package higgsfield

// RequestStatus values returned by the Higgsfield API.
const (
	StatusQueued     = "queued"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusNSFW       = "nsfw"
	StatusCanceled   = "canceled"
)

// ImageRequest is the JSON body for Soul-style text-to-image endpoints.
type ImageRequest struct {
	Prompt      string `json:"prompt"`
	AspectRatio string `json:"aspect_ratio"`
	Resolution  string `json:"resolution"`
}

// SubmitResponse is returned immediately after a successful submission.
type SubmitResponse struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	StatusURL string `json:"status_url"`
	CancelURL string `json:"cancel_url"`
}

// MediaOutput holds a downloadable artifact URL.
type MediaOutput struct {
	URL string `json:"url"`
}

// RequestStatus is the current state of a generation request.
type RequestStatus struct {
	Status    string        `json:"status"`
	RequestID string        `json:"request_id"`
	StatusURL string        `json:"status_url,omitempty"`
	CancelURL string        `json:"cancel_url,omitempty"`
	Error     *string       `json:"error"`
	Images    []MediaOutput `json:"images,omitempty"`
	Video     *MediaOutput  `json:"video,omitempty"`
}

// Terminal reports whether the request reached a final state.
func (s RequestStatus) Terminal() bool {
	switch s.Status {
	case StatusCompleted, StatusFailed, StatusNSFW, StatusCanceled:
		return true
	default:
		return false
	}
}

// FirstImageURL returns the first image URL when status is completed.
func (s RequestStatus) FirstImageURL() (string, bool) {
	if len(s.Images) == 0 || s.Images[0].URL == "" {
		return "", false
	}
	return s.Images[0].URL, true
}
