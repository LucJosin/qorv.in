package middlewarex

import (
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/lucjosin/qorv.in/internal/slogx"
)

// Logger attaches request fields to the context logger and logs the response.
// Register it after RequestID and TraceID and before panic recovery.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := slogx.FromCtx(r.Context()).With(
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		params := queryParams(r)
		if len(params) > 0 {
			log = log.With("params", params)
		}

		requestID := RequestIDFromContext(r.Context())
		if requestID != "" {
			log = log.With("request_id", requestID)
		}
		traceID := TraceIDFromContext(r.Context())
		if traceID != "" {
			log = log.With("trace_id", traceID)
		}
		ctx := slogx.WithContext(r.Context(), log)
		r = r.WithContext(ctx)
		writer := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()

		next.ServeHTTP(writer, r)

		status := writer.Status()
		if status == 0 {
			status = http.StatusOK
		}
		level := slog.LevelInfo
		if status >= http.StatusInternalServerError {
			level = slog.LevelError
		} else if status >= http.StatusBadRequest {
			level = slog.LevelWarn
		}

		log.Log(ctx, level, "request handled",
			"status", status,
			"bytes", writer.BytesWritten(),
			"duration", time.Since(start),
		)
	})
}

func queryParams(r *http.Request) url.Values {
	sensitiveParams := []string{
		"token",
		"password",
		"authorization",
		"secret",
		"api_key",
		"apikey",
		"access_token",
		"refresh_token",
		"client_secret",
		"id_token",
	}
	params := r.URL.Query()
	for name := range params {
		lowerName := strings.ToLower(name)
		sensitive := slices.Contains(sensitiveParams, lowerName)
		if sensitive {
			params.Del(name)
		}
	}
	return params
}
