package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/lucjosin/qorv.in/internal/httpx"
	"github.com/lucjosin/qorv.in/internal/pagination"
	"github.com/lucjosin/qorv.in/internal/slogx"
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
			Errors:     fields,
			Path:       r.URL.Path,
			Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		},
	}
	httpx.WriteJSON(r.Context(), w, status, response)
}
