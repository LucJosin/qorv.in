package middlewarex

import (
	"context"
	"net/http"
	"uuid"
)

type traceIDKey struct{}

// TraceID generates an internal UUIDv7 for each request, independent of client IDs.
// It is attached to the context for logging and is not returned in a response header.
func TraceID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := uuid.NewV7().String()
		ctx := context.WithValue(r.Context(), traceIDKey{}, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TraceIDFromContext returns the internal trace ID, or an empty string when absent.
func TraceIDFromContext(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDKey{}).(string)
	return traceID
}
