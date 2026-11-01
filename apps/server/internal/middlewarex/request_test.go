package middlewarex_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/middlewarex"
)

func TestRequestID_Values(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name: "generated",
		},
		{
			name:  "supplied",
			value: "client-request-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set(middlewarex.RequestIDHeader, tt.value)
			recorder := httptest.NewRecorder()
			var contextID string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contextID = middlewarex.RequestIDFromContext(r.Context())
				w.WriteHeader(http.StatusNoContent)
			})
			handler := middlewarex.RequestID(next)

			handler.ServeHTTP(recorder, request)

			want := contextID
			got := recorder.Header().Get(middlewarex.RequestIDHeader)
			if got == "" {
				t.Fatal("missing request ID")
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("header/context mismatch (-want +got):\n%s", diff)
			}
			if tt.value == "" {
				assertUUIDv7(t, got)
			} else {
				want = tt.value
				if diff := cmp.Diff(want, got); diff != "" {
					t.Fatalf("supplied ID mismatch (-want +got):\n%s", diff)
				}
			}
			wantStatus := http.StatusNoContent
			gotStatus := recorder.Code
			if diff := cmp.Diff(wantStatus, gotStatus); diff != "" {
				t.Fatalf("status mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRequestID_UniqueValues(t *testing.T) {
	ids := make(map[string]bool)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := middlewarex.RequestIDFromContext(r.Context())
		ids[id] = true
		w.WriteHeader(http.StatusNoContent)
	})
	handler := middlewarex.RequestID(next)

	for range 2 {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
	}

	want := 2
	got := len(ids)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unique ID count mismatch (-want +got):\n%s", diff)
	}
}

func TestRequestIDFromContext_Absent(t *testing.T) {
	want := ""
	got := middlewarex.RequestIDFromContext(t.Context())
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("request ID mismatch (-want +got):\n%s", diff)
	}
}

func assertUUIDv7(t *testing.T, value string) {
	t.Helper()

	id, err := uuid.Parse(value)
	if err != nil {
		t.Fatal(err)
	}

	validVersion := id[6]>>4 == 7 && id[8]&0xC0 == 0x80
	if !validVersion {
		t.Fatalf("ID %q is not a UUIDv7", value)
	}
}
