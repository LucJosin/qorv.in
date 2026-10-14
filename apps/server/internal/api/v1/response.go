package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lucjosin/qorv.in/internal/httpx"
	"github.com/lucjosin/qorv.in/internal/pagination"
	"github.com/lucjosin/qorv.in/internal/slogx"
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

// APIResponse represents a standardized API response format,
// containing a data payload of any type and additional metadata.
type APIResponse[T any] struct {
	Data       T                    `json:"data"`
	Pagination *pagination.Metadata `json:"pagination,omitempty"`
}

type APIErrorResponse struct {
	Error ErrorResponse `json:"error"`
}

// internalError logs internal details and returns a generic JSON error.
func internalError(w http.ResponseWriter, r *http.Request, err error) {
	slogx.FromCtx(r.Context()).Error("server error", "error", err)
	writeError(w, r, http.StatusInternalServerError, "An internal server error occurred.", nil)
}

// writeValidationError writes field validation failures as bad requests.
//
// Unexpected errors or validation errors without fields fall back to internalError.
func writeValidationError(w http.ResponseWriter, r *http.Request, err error, message string) {
	validation, ok := errors.AsType[*validatorx.ValidationError](err)
	if !ok || validation == nil || len(validation.Fields) == 0 {
		internalError(w, r, err)
		return
	}

	fields := make([]FieldError, 0, len(validation.Fields))
	for _, field := range validation.Fields {
		key := FieldInvalid
		detail := field.Message
		param := strings.Join(field.Params, ",")
		switch field.Tag {
		case "required":
			key = FieldRequired
			if detail == "" {
				detail = "is required"
			}
		case "min":
			key = FieldMin
			if detail == "" {
				detail = "must be at least " + param
			}
		case "max":
			key = FieldMax
			if detail == "" {
				detail = "must be at most " + param
			}
		default:
			if detail == "" {
				detail = "has an invalid value"
			}
		}

		detailError := FieldError{
			Field:   field.Field,
			Key:     key,
			Message: fmt.Sprintf("Field %s %s", field.Field, detail),
		}
		fields = append(fields, detailError)
	}

	response := APIErrorResponse{
		Error: ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Status:     http.StatusText(http.StatusBadRequest),
			Message:    message,
			Details:    fields,
			Path:       r.URL.Path,
			Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		},
	}
	httpx.WriteJSON(r.Context(), w, http.StatusBadRequest, response)
}

// writeResponse sends a JSON response with the specified status code and body.
//
// It wraps the body in an APIResponse and includes pagination metadata when present.
func writeResponse[T any](ctx context.Context, w http.ResponseWriter, status int, body T, metadata *pagination.Metadata) {
	response := APIResponse[T]{
		Data:       body,
		Pagination: metadata,
	}
	httpx.WriteJSON(ctx, w, status, response)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, message string, fields []FieldError) {
	response := APIErrorResponse{
		Error: ErrorResponse{
			StatusCode: status,
			Status:     http.StatusText(status),
			Message:    message,
			Details:    fields,
			Path:       r.URL.Path,
			Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		},
	}
	httpx.WriteJSON(r.Context(), w, status, response)
}
