package vnish

import (
	"errors"
	"fmt"
)

var (
	// ErrUnauthorized indicates authentication is required or token is invalid.
	ErrUnauthorized = errors.New("unauthorized: authentication required")

	// ErrForbidden indicates the request was forbidden (invalid API key).
	ErrForbidden = errors.New("forbidden: invalid or missing API key")

	// ErrNotFound indicates the requested resource was not found.
	ErrNotFound = errors.New("not found")

	// ErrConflict indicates a conflict with the current state.
	ErrConflict = errors.New("conflict")

	// ErrMinerFailure indicates the miner is in a failure state.
	ErrMinerFailure = errors.New("miner is in failure state")

	// ErrAuthenticationFailed indicates authentication failed.
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrAPIKeyRequired indicates an API key is required for this operation.
	ErrAPIKeyRequired = errors.New("API key required for this operation")
)

// APIError represents an error returned by the VNish API.
type APIError struct {
	StatusCode int
	Message    string
	Endpoint   string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("vnish API error (HTTP %d) at %s: %s", e.StatusCode, e.Endpoint, e.Message)
	}
	return fmt.Sprintf("vnish API error (HTTP %d) at %s", e.StatusCode, e.Endpoint)
}

// IsUnauthorized returns true if the error indicates authentication is needed.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

// IsForbidden returns true if the error indicates an API key issue.
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == 403
}

// IsNotFound returns true if the resource was not found.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsAuthError returns true if authentication or authorization failed.
func IsAuthError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsUnauthorized() || apiErr.IsForbidden()
	}
	return errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden)
}

// NeedsAPIKey returns true if the error indicates an API key is needed.
func NeedsAPIKey(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsForbidden()
	}
	return errors.Is(err, ErrForbidden) || errors.Is(err, ErrAPIKeyRequired)
}
