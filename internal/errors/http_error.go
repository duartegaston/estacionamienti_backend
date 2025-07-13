package errors

import "net/http"

// HTTPError represents an error with an associated HTTP status code.
type HTTPError struct {
	Code    int
	Message string
}

func (e *HTTPError) Error() string {
	return e.Message
}

// NewHTTPError creates a new HTTPError with the given code and message.
func NewHTTPError(code int, message string) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
	}
}

// Helper for common errors
var (
	ErrUnauthorized = func(msg string) *HTTPError { return NewHTTPError(http.StatusUnauthorized, msg) }
)
