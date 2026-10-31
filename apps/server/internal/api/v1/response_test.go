package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestWriteResponse_WithoutPagination(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := APIResponse[[]string]{
		Data: []string{"collection"},
	}

	writeResponse(context.Background(), recorder, http.StatusOK, body)

	wantStatus := http.StatusOK
	gotStatus := recorder.Code
	if diff := cmp.Diff(wantStatus, gotStatus); diff != "" {
		t.Fatalf("status mismatch (-want +got):\n%s", diff)
	}
	var response map[string]json.RawMessage
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]json.RawMessage{
		"data": json.RawMessage(`["collection"]`),
	}
	got := response
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("response mismatch (-want +got):\n%s", diff)
	}
}

func TestWriteError_StatusResponses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
	}{
		{
			name:    "forbidden",
			status:  http.StatusForbidden,
			message: "Access denied",
		},
		{
			name:    "not found",
			status:  http.StatusNotFound,
			message: "Collection not found",
		},
		{
			name:   "conflict without message",
			status: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/collections/", nil)

			writeError(recorder, request, tt.status, tt.message, nil)

			wantStatus := tt.status
			gotStatus := recorder.Code
			if diff := cmp.Diff(wantStatus, gotStatus); diff != "" {
				t.Fatalf("status mismatch (-want +got):\n%s", diff)
			}
			var response APIErrorResponse
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			if err != nil {
				t.Fatal(err)
			}
			_, err = time.Parse(time.RFC3339Nano, response.Error.Timestamp)
			if err != nil {
				t.Fatalf("invalid timestamp: %v", err)
			}

			want := ErrorResponse{
				StatusCode: tt.status,
				Status:     http.StatusText(tt.status),
				Message:    tt.message,
				Path:       request.URL.Path,
			}
			got := response.Error
			got.Timestamp = ""
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
