package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lucjosin/qorv.in/internal/httpx"
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

// APIResponse represents a standardized API response format,
// containing a data payload of any type and additional metadata.
type APIResponse[T any] struct {
	Data  T              `json:"data,omitzero"`
	Error *ErrorResponse `json:"error,omitzero"`
}

// APIErrorResponse wraps an API error without a success payload.
type APIErrorResponse struct {
	Error ErrorResponse `json:"error"`
}

// internalError sends an [http.StatusInternalServerError] response and logs the error.
//
// No body is sent to the client.
func internalError(ctx context.Context, w http.ResponseWriter, err error) {
	httpx.WriteErrorf(ctx, w, err, http.StatusInternalServerError, nil)
}

// writeValidationError writes field validation failures as bad requests.
//
// Unexpected errors or validation errors without fields fall back to internalError.
func writeValidationError(w http.ResponseWriter, r *http.Request, err error, message string) {
	validation, ok := errors.AsType[*validatorx.ValidationError](err)
	if !ok || validation == nil || len(validation.Fields) == 0 {
		internalError(r.Context(), w, err)
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
		fields = append(fields, FieldError{
			Field:   field.Field,
			Key:     key,
			Message: fmt.Sprintf("Field %s %s", field.Field, detail),
		})
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
// It wraps the body in an APIResponse struct and includes pagination metadata if provided.
func writeResponse[T any](ctx context.Context, w http.ResponseWriter, status int, body T) {
	httpx.WriteJSON(ctx, w, status, APIResponse[T]{
		Data: body,
	})
}
