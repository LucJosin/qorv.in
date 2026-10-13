package pagination_test

import (
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/LucJosin/opid"
	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/errs"
	"github.com/lucjosin/qorv.in/internal/pagination"
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query, err := url.ParseQuery(test.query)
			if err != nil {
				t.Fatal(err)
			}
			got, err := pagination.Parse(query, filters, codec)
			if err != nil {
				t.Fatal(err)
			}
			want := test.want
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("request mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParse_InvalidQueryParameters(t *testing.T) {
	codec := newCodec(t)
	filters := Filters{
		Resource: "collections",
	}
	tests := []struct {
		name  string
		query string
		field string
	}{
		{
			name:  "zero limit",
			query: "limit=0",
			field: "limit",
		},
		{
			name:  "negative limit",
			query: "limit=-1",
			field: "limit",
		},
		{
			name:  "large limit",
			query: "limit=101",
			field: "limit",
		},
		{
			name:  "non-integer limit",
			query: "limit=1.5",
			field: "limit",
		},
		{
			name:  "overflow limit",
			query: "limit=999999999999999999999999",
			field: "limit",
		},
		{
			name:  "empty limit",
			query: "limit=",
			field: "limit",
		},
		{
			name:  "duplicate limit",
			query: "limit=1&limit=2",
			field: "limit",
		},
		{
			name:  "offset",
			query: "offset=10",
			field: "offset",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query, err := url.ParseQuery(test.query)
			if err != nil {
				t.Fatal(err)
			}
			_, err = pagination.Parse(query, filters, codec)
			var validation *errs.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("error = %v, want a validation error", err)
			}
			got := validation.Field
			want := test.field
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("field mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParse_InvalidCursors(t *testing.T) {
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
		field string
	}{
		{
			name:  "empty cursor",
			query: "cursor=",
			field: "cursor",
		},
		{
			name:  "duplicate cursor",
			query: "cursor=a&cursor=b",
			field: "cursor",
		},
		{
			name:  "malformed cursor",
			query: "cursor=invalid",
			field: "cursor",
		},
		{
			name:  "oversize cursor",
			query: "cursor=" + strings.Repeat("a", 2049),
			field: "cursor",
		},
		{
			name:  "wrong resource",
			query: "cursor=" + wrongResource,
			field: "cursor",
		},
		{
			name:  "wrong workspace filter",
			query: "cursor=" + wrongWorkspace,
			field: "cursor",
		},
		{
			name:  "wrong prefix",
			query: "cursor=" + wrongPrefix,
			field: "cursor",
		},
		{
			name:  "unsupported version",
			query: "cursor=" + rawCursor(`{"version":2}`),
			field: "cursor",
		},
		{
			name:  "unknown field",
			query: "cursor=" + rawCursor(`{"unknown":true}`),
			field: "cursor",
		},
		{
			name:  "trailing JSON",
			query: "cursor=" + rawCursor(`{} {}`),
			field: "cursor",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query, err := url.ParseQuery(test.query)
			if err != nil {
				t.Fatal(err)
			}
			_, err = pagination.Parse(query, filters, codec)
			var validation *errs.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("error = %v, want a validation error", err)
			}
			got := validation.Field
			want := test.field
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("field mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewPage_Boundaries(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  pagination.Page[int]
	}{
		{
			name: "empty",
			want: pagination.Page[int]{
				Items: []int{},
			},
		},
		{
			name:  "exact limit",
			items: []int{3, 2},
			want: pagination.Page[int]{
				Items: []int{3, 2},
			},
		},
		{
			name:  "extra row",
			items: []int{3, 2, 1},
			want: pagination.Page[int]{
				Items:   []int{3, 2},
				HasMore: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := pagination.NewPage(test.items, 2)
			want := test.want
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("page mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func newCodec(t *testing.T) *opid.Codec {
	t.Helper()
	codec, err := opid.NewCodec("col")
	if err != nil {
		t.Fatal(err)
	}
	return codec
}

func rawCursor(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

// Filters is a test-specific filter type; pagination does not own domain filters.
type Filters struct {
	Resource    string  `json:"resource"`
	WorkspaceID opid.ID `json:"workspace_id,omitzero"`
}
