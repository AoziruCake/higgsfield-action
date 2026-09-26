package higgsfield

import "fmt"

// APIError is a non-success HTTP response from the Higgsfield API.
type APIError struct {
	StatusCode int
	Detail     string
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("higgsfield api: %s (HTTP %d)", e.Detail, e.StatusCode)
	}
	return fmt.Sprintf("higgsfield api: HTTP %d", e.StatusCode)
}

// IsUnauthorized reports authentication failures.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

// IsNotFound reports missing requests or resources.
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
