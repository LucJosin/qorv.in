package pagination_test

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/LucJosin/opid"
	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/pagination"
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

func TestParse_ValidRequests(t *testing.T) {
	codec := newCodec(t)
	filters := Filters{
		Resource: "collections",
	}

	position := pagination.Cursor{
		CreatedAt: time.Date(2026, 10, 12, 12, 0, 0, 123000, time.UTC),
		ID:        uuid.NewV7(),
	}

	cursor, err := pagination.Encode(position, filters, codec)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		query string
		want  pagination.Query
	}{
		{
			name: "default",
			want: pagination.Query{
				Limit: pagination.DefaultLimit,
			},
		},
		{
			name:  "minimum",
			query: "limit=1",
			want: pagination.Query{
				Limit: 1,
			},
		},
		{
			name:  "maximum",
			query: "limit=100",
			want: pagination.Query{
				Limit: pagination.MaxLimit,
			},
		},
		{
			name:  "continuation",
			query: "limit=2&cursor=" + cursor,
			want: pagination.Query{
				Limit: 2,
				After: &position,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatal(err)
			}

			got, err := pagination.Parse(query, filters, codec)
			if err != nil {
				t.Fatal(err)
			}

			want := tt.want
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("request mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParse_InvalidRequests(t *testing.T) {
	codec := newCodec(t)
	filters := Filters{
		Resource: "collections",
	}

	position := pagination.Cursor{
		CreatedAt: time.Now().UTC(),
		ID:        uuid.NewV7(),
	}

	otherFilters := Filters{
		Resource: "workspaces",
	}

	wrongResource, err := pagination.Encode(position, otherFilters, codec)
	if err != nil {
		t.Fatal(err)
	}

	wrongCodec, err := opid.NewCodec("ws")
	if err != nil {
		t.Fatal(err)
	}

	wrongPrefix, err := pagination.Encode(position, filters, wrongCodec)
	if err != nil {
		t.Fatal(err)
	}

	workspaceID, err := wrongCodec.Encode(uuid.NewV7())
	if err != nil {
		t.Fatal(err)
	}

	workspaceFilters := Filters{
		Resource:    "collections",
		WorkspaceID: workspaceID,
	}

	wrongWorkspace, err := pagination.Encode(position, workspaceFilters, codec)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		query string
		want  []validatorx.FieldError
	}{
		{
			name:  "zero limit",
			query: "limit=0",
			want: []validatorx.FieldError{
				validatorx.Field("limit", "must be between 1 and 100"),
			},
		},
		{
			name:  "negative limit",
			query: "limit=-1",
			want: []validatorx.FieldError{
				validatorx.Field("limit", "must be between 1 and 100"),
			},
		},
		{
			name:  "large limit",
			query: "limit=101",
			want: []validatorx.FieldError{
				validatorx.Field("limit", "must be between 1 and 100"),
			},
		},
		{
			name:  "non-integer limit",
			query: "limit=1.5",
			want: []validatorx.FieldError{
				validatorx.Field("limit", "must be an integer between 1 and 100"),
			},
		},
		{
			name:  "overflow limit",
			query: "limit=999999999999999999999999",
			want: []validatorx.FieldError{
				validatorx.Field("limit", "must be an integer between 1 and 100"),
			},
		},
		{
			name:  "empty limit",
			query: "limit=",
			want: []validatorx.FieldError{
				validatorx.Field("limit", "must have exactly one non-empty value"),
			},
		},
		{
			name:  "duplicate limit",
			query: "limit=1&limit=2",
			want: []validatorx.FieldError{
				validatorx.Field("limit", "must have exactly one non-empty value"),
			},
		},
		{
			name:  "empty cursor",
			query: "cursor=",
			want: []validatorx.FieldError{
				validatorx.Field("cursor", "must have exactly one non-empty value"),
			},
		},
		{
			name:  "duplicate cursor",
			query: "cursor=a&cursor=b",
			want: []validatorx.FieldError{
				validatorx.Field("cursor", "must have exactly one non-empty value"),
			},
		},
		{
			name:  "malformed cursor",
			query: "cursor=invalid",
			want: []validatorx.FieldError{
				validatorx.Field("cursor", "invalid or unsupported cursor"),
			},
		},
		{
			name:  "oversize cursor",
			query: "cursor=" + strings.Repeat("a", 2049),
			want: []validatorx.FieldError{
				validatorx.Field("cursor", "invalid or unsupported cursor"),
			},
		},
		{
			name:  "wrong resource",
			query: "cursor=" + wrongResource,
			want: []validatorx.FieldError{
				validatorx.Field("cursor", "invalid or unsupported cursor"),
			},
		},
		{
			name:  "wrong workspace filter",
			query: "cursor=" + wrongWorkspace,
			want: []validatorx.FieldError{
				validatorx.Field("cursor", "invalid or unsupported cursor"),
			},
		},
		{
			name:  "wrong prefix",
			query: "cursor=" + wrongPrefix,
			want: []validatorx.FieldError{
				validatorx.Field("cursor", "invalid or unsupported cursor"),
			},
		},
		{
			name:  "unsupported version",
			query: "cursor=" + rawCursor(`{"version":2}`),
			want: []validatorx.FieldError{
				validatorx.Field("cursor", "invalid or unsupported cursor"),
			},
		},
		{
			name:  "unknown field",
			query: "cursor=" + rawCursor(`{"unknown":true}`),
			want: []validatorx.FieldError{
				validatorx.Field("cursor", "invalid or unsupported cursor"),
			},
		},
		{
			name:  "trailing JSON",
			query: "cursor=" + rawCursor(`{} {}`),
			want: []validatorx.FieldError{
				validatorx.Field("cursor", "invalid or unsupported cursor"),
			},
		},
		{
			name:  "conversion and validation errors",
			query: "limit=101&cursor=invalid",
			want: []validatorx.FieldError{
				validatorx.Field("cursor", "invalid or unsupported cursor"),
				validatorx.Field("limit", "must be between 1 and 100"),
			},
		},
		{
			name:  "conversion errors without duplicates",
			query: "limit=invalid&cursor=invalid",
			want: []validatorx.FieldError{
				validatorx.Field("limit", "must be an integer between 1 and 100"),
				validatorx.Field("cursor", "invalid or unsupported cursor"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatal(err)
			}

			gotQuery, err := pagination.Parse(query, filters, codec)

			validation, ok := errors.AsType[*validatorx.ValidationError](err)
			if !ok {
				t.Fatalf("error = %v, want a validation error", err)
			}

			want := tt.want
			got := validation.Fields
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("fields mismatch (-want +got):\n%s", diff)
			}
			wantQuery := pagination.Query{}
			if diff := cmp.Diff(wantQuery, gotQuery); diff != "" {
				t.Fatalf("query mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParse_DomainParameters(t *testing.T) {
	codec := newCodec(t)
	filters := Filters{
		Resource: "collections",
	}

	query := url.Values{
		"limit":        {"2"},
		"workspace_id": {"workspace"},
	}

	wantValues := url.Values{
		"limit":        {"2"},
		"workspace_id": {"workspace"},
	}

	got, err := pagination.Parse(query, filters, codec)
	if err != nil {
		t.Fatal(err)
	}

	want := pagination.Query{
		Limit: 2,
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("query mismatch (-want +got):\n%s", diff)
	}

	gotValues := query
	if diff := cmp.Diff(wantValues, gotValues); diff != "" {
		t.Fatalf("input changed (-want +got):\n%s", diff)
	}
}
