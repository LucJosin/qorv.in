package v1

import (
	"net/http"
)

// Field validation codes form the public API contract.
const (
	FieldRequired = "FIELD_REQUIRED"
	FieldInvalid  = "FIELD_INVALID"
	FieldMin      = "FIELD_MIN"
	FieldMax      = "FIELD_MAX"
)

// FieldError represents a validation error for a specific field
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Key     string `json:"key"`
}

// ErrorResponse represents a standard API error response
type ErrorResponse struct {
	StatusCode int          `json:"status_code"`
	Status     string       `json:"status"`
	Message    string       `json:"message,omitempty"`
	Details    []FieldError `json:"details,omitempty"`

	Path      string `json:"path,omitempty"`
	Timestamp string `json:"timestamp"`
}

func (e *ErrorResponse) Error() string {
	switch {
	case e.Message != "":
		return e.Message
	case e.StatusCode != 0:
		return http.StatusText(e.StatusCode)
	default:
		return "unknown error"
	}
}
