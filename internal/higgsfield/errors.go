package higgsfield

import "fmt"

// APIError is a non-success HTTP response from the Higgsfield API.
type APIError struct {
	StatusCode int
	Detail     string // API "detail" field when present
}

func (e *APIError) Error() string {
	if e.StatusCode == 403 && e.Detail == "not_enough_credits" {
		return "higgsfield api: account has no remaining credits (HTTP 403)"
	}
	if e.Detail != "" {
		return fmt.Sprintf("higgsfield api: %s (HTTP %d)", e.Detail, e.StatusCode)
	}
	return fmt.Sprintf("higgsfield api: HTTP %d", e.StatusCode)
}

// IsUnauthorized reports authentication failures (do not retry).
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

// IsNotFound reports a missing request or resource (do not retry).
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// StatusError is returned when polling reaches a non-success terminal state.
type StatusError struct {
	Status RequestStatus
}

func (e *StatusError) Error() string {
	return e.Status.FailureMessage()
}
