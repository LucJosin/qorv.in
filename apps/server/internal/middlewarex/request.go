package middlewarex

import (
	"context"
	"net/http"
	"uuid"
)

type requestIDKey struct{}

// RequestIDHeader identifies the request in client and server logs.
const RequestIDHeader = "X-Request-ID"

// RequestID preserves a supplied request ID or generates a UUIDv7.
// The ID is attached to the context and returned in the response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.NewV7().String()
		}
		w.Header().Set(RequestIDHeader, requestID)
		ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext returns the request ID, or an empty string when absent.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}
