package middlewarex_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/middlewarex"
	"github.com/lucjosin/qorv.in/internal/slogx"
)

func TestLogger_Response(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantStatus int
		wantLevel  string
	}{
		{
			name:       "empty response defaults to OK",
			wantStatus: http.StatusOK,
			wantLevel:  "INFO",
		},
		{
			name:       "implicit OK with byte count",
			body:       "café",
			wantStatus: http.StatusOK,
			wantLevel:  "INFO",
		},
		{
			name:       "created",
			status:     http.StatusCreated,
			body:       "created",
			wantStatus: http.StatusCreated,
			wantLevel:  "INFO",
		},
		{
			name:       "no content",
			status:     http.StatusNoContent,
			wantStatus: http.StatusNoContent,
			wantLevel:  "INFO",
		},
		{
			name:       "redirect",
			status:     http.StatusFound,
			wantStatus: http.StatusFound,
			wantLevel:  "INFO",
		},
		{
			name:       "client error",
			status:     http.StatusBadRequest,
			body:       "invalid request",
			wantStatus: http.StatusBadRequest,
			wantLevel:  "WARN",
		},
		{
			name:       "server error",
			status:     http.StatusInternalServerError,
			wantStatus: http.StatusInternalServerError,
			wantLevel:  "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, output := loggerRequest(http.MethodGet, "/collections/", nil)
			recorder := httptest.NewRecorder()
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.status != 0 {
					w.WriteHeader(tt.status)
				}
				if tt.body != "" {
					_, err := io.WriteString(w, tt.body)
					if err != nil {
						t.Fatal(err)
					}
				}
			})
			handler := middlewarex.Logger(next)

			handler.ServeHTTP(recorder, request)

			wantStatus := tt.wantStatus
			gotStatus := recorder.Code
			if diff := cmp.Diff(wantStatus, gotStatus); diff != "" {
				t.Fatalf("status mismatch (-want +got):\n%s", diff)
			}
			wantBody := tt.body
			gotBody := recorder.Body.String()
			if diff := cmp.Diff(wantBody, gotBody); diff != "" {
				t.Fatalf("body mismatch (-want +got):\n%s", diff)
			}
			records := readRequestLogs(t, output)
			wantCount := 1
			gotCount := len(records)
			if diff := cmp.Diff(wantCount, gotCount); diff != "" {
				t.Fatalf("log count mismatch (-want +got):\n%s", diff)
			}
			got := records[0]
			if got.Duration < 0 {
				t.Fatal("negative request duration")
			}
			got.Duration = 0
			want := requestLog{
				Level:      tt.wantLevel,
				Message:    "request handled",
				Component:  "api",
				Method:     http.MethodGet,
				Path:       "/collections/",
				RemoteAddr: request.RemoteAddr,
				Status:     tt.wantStatus,
				Bytes:      len(tt.body),
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("log mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLogger_RequestContext(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
	}{
		{
			name: "generated request ID",
		},
		{
			name:      "supplied request ID",
			requestID: "request-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, output := loggerRequest(http.MethodPost, "/collections/", nil)
			request.Header.Set(middlewarex.RequestIDHeader, tt.requestID)
			recorder := httptest.NewRecorder()
			var traceID string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				log := slogx.FromCtx(r.Context())
				traceID = middlewarex.TraceIDFromContext(r.Context())
				log.InfoContext(r.Context(), "inside handler")
				w.WriteHeader(http.StatusNoContent)
			})
			handler := middlewarex.RequestID(middlewarex.TraceID(middlewarex.Logger(next)))

			handler.ServeHTTP(recorder, request)

			requestID := recorder.Header().Get(middlewarex.RequestIDHeader)
			if requestID == "" {
				t.Fatal("missing response request ID")
			}
			if traceID == "" {
				t.Fatal("missing trace ID in request context")
			}
			if tt.requestID != "" {
				want := tt.requestID
				got := requestID
				if diff := cmp.Diff(want, got); diff != "" {
					t.Fatalf("request ID mismatch (-want +got):\n%s", diff)
				}
			}
			records := readRequestLogs(t, output)
			wantCount := 2
			gotCount := len(records)
			if diff := cmp.Diff(wantCount, gotCount); diff != "" {
				t.Fatalf("log count mismatch (-want +got):\n%s", diff)
			}
			for _, record := range records {
				want := requestLog{
					Component:  "api",
					Method:     http.MethodPost,
					Path:       "/collections/",
					RemoteAddr: request.RemoteAddr,
					RequestID:  requestID,
					TraceID:    traceID,
				}
				got := requestLog{
					Component:  record.Component,
					Method:     record.Method,
					Path:       record.Path,
					RemoteAddr: record.RemoteAddr,
					RequestID:  record.RequestID,
					TraceID:    record.TraceID,
				}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Fatalf("context mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestLogger_RecoveredPanic(t *testing.T) {
	request, output := loggerRequest(http.MethodGet, "/panic", nil)
	recorder := httptest.NewRecorder()
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("test panic")
	})
	handler := middlewarex.Logger(middleware.Recoverer(next))

	handler.ServeHTTP(recorder, request)

	wantStatus := http.StatusInternalServerError
	gotStatus := recorder.Code
	if diff := cmp.Diff(wantStatus, gotStatus); diff != "" {
		t.Fatalf("status mismatch (-want +got):\n%s", diff)
	}
	records := readRequestLogs(t, output)
	wantCount := 1
	gotCount := len(records)
	if diff := cmp.Diff(wantCount, gotCount); diff != "" {
		t.Fatalf("log count mismatch (-want +got):\n%s", diff)
	}
	want := requestLog{
		Level:  "ERROR",
		Status: http.StatusInternalServerError,
	}
	got := requestLog{
		Level:  records[0].Level,
		Status: records[0].Status,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("panic log mismatch (-want +got):\n%s", diff)
	}
}

func TestLogger_Flushing(t *testing.T) {
	request, output := loggerRequest(http.MethodGet, "/stream", nil)
	recorder := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer lost its flushing capability")
		}
		_, err := io.WriteString(w, "stream")
		if err != nil {
			t.Fatal(err)
		}
		flusher.Flush()
	})
	handler := middlewarex.Logger(next)

	handler.ServeHTTP(recorder, request)

	if !recorder.Flushed {
		t.Fatal("response was not flushed")
	}
	records := readRequestLogs(t, output)
	want := 6
	got := records[0].Bytes
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("bytes mismatch (-want +got):\n%s", diff)
	}
}

func TestLogger_QueryParameters(t *testing.T) {
	values := url.Values{
		"limit":         {"20"},
		"category":      {"Design", "Tools"},
		"token":         {"token-secret", "other-token-secret"},
		"PASSWORD":      {"password-secret"},
		"Authorization": {"authorization-secret"},
		"secret":        {"secret-value"},
		"api_key":       {"api-key-secret"},
		"apiKey":        {"api-key-secret"},
		"access_token":  {"access-secret"},
		"refresh_token": {"refresh-secret"},
		"client_secret": {"client-secret"},
		"id_token":      {"id-secret"},
	}
	request, output := loggerRequest(http.MethodGet, "/collections/?"+values.Encode(), nil)
	recorder := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := slogx.FromCtx(r.Context())

		want := values
		got := r.URL.Query()
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("request query mismatch (-want +got):\n%s", diff)
		}
		log.InfoContext(r.Context(), "inside handler")
		w.WriteHeader(http.StatusNoContent)
	})
	handler := middlewarex.Logger(next)

	handler.ServeHTTP(recorder, request)

	containsSecret := strings.Contains(output.String(), "secret")
	if containsSecret {
		t.Fatalf("log contains sensitive query data: %s", output.String())
	}
	records := readRequestLogs(t, output)
	wantCount := 2
	gotCount := len(records)
	if diff := cmp.Diff(wantCount, gotCount); diff != "" {
		t.Fatalf("log count mismatch (-want +got):\n%s", diff)
	}
	want := url.Values{
		"limit":    {"20"},
		"category": {"Design", "Tools"},
	}
	for _, record := range records {
		got := record.Params
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("logged query mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestLogger_RequestData(t *testing.T) {
	body := `{"password":"body-secret"}`
	request, output := loggerRequest(http.MethodPost, "/collections/?token=query-secret", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer header-secret")
	recorder := httptest.NewRecorder()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		want := body
		got := string(data)
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("request body mismatch (-want +got):\n%s", diff)
		}
		want = "query-secret"
		got = r.URL.Query().Get("token")
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("query mismatch (-want +got):\n%s", diff)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handler := middlewarex.Logger(next)

	handler.ServeHTTP(recorder, request)

	containsSecret := strings.Contains(output.String(), "secret")
	if containsSecret {
		t.Fatalf("log contains request data: %s", output.String())
	}
}

type requestLog struct {
	Level      string     `json:"level"`
	Message    string     `json:"msg"`
	Component  string     `json:"component"`
	Method     string     `json:"method"`
	Path       string     `json:"path"`
	RemoteAddr string     `json:"remote_addr"`
	RequestID  string     `json:"request_id"`
	TraceID    string     `json:"trace_id"`
	Params     url.Values `json:"params"`
	Status     int        `json:"status"`
	Bytes      int        `json:"bytes"`
	Duration   int64      `json:"duration"`
}

func loggerRequest(method, target string, body io.Reader) (*http.Request, *bytes.Buffer) {
	output := &bytes.Buffer{}
	handler := slog.NewJSONHandler(output, nil)
	log := &slogx.Logger{
		Logger: slog.New(handler).With("component", "api"),
	}
	request := httptest.NewRequest(method, target, body)
	ctx := slogx.WithContext(request.Context(), log)
	return request.WithContext(ctx), output
}

func readRequestLogs(t *testing.T, output *bytes.Buffer) []requestLog {
	t.Helper()

	var records []requestLog
	decoder := json.NewDecoder(output)
	for {
		var record requestLog
		err := decoder.Decode(&record)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}
