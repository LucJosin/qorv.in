package middlewarex_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/middlewarex"
)

func TestTraceID_InternalValues(t *testing.T) {
	ids := make(map[string]bool)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := middlewarex.TraceIDFromContext(r.Context())
		assertUUIDv7(t, id)
		ids[id] = true

		requestID := middlewarex.RequestIDFromContext(r.Context())
		want := "client-request"
		got := requestID
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("request ID mismatch (-want +got):\n%s", diff)
		}
		if id == requestID {
			t.Fatal("internal trace ID reused the client request ID")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handler := middlewarex.RequestID(middlewarex.TraceID(next))

	for range 2 {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set(middlewarex.RequestIDHeader, "client-request")
		request.Header.Set("X-Trace-ID", "client-trace")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		want := ""
		got := recorder.Header().Get("X-Trace-ID")
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("trace response header mismatch (-want +got):\n%s", diff)
		}
	}

	want := 2
	got := len(ids)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unique trace count mismatch (-want +got):\n%s", diff)
	}
}

func TestTraceIDFromContext_Absent(t *testing.T) {
	want := ""
	got := middlewarex.TraceIDFromContext(t.Context())
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("trace ID mismatch (-want +got):\n%s", diff)
	}
}
