package pagination_test

import (
	"errors"
	"net/url"
	"testing"
	"time"
	"uuid"

	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/errs"
	"github.com/lucjosin/qorv.in/internal/pagination"
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

func TestNewQuery_Sort(t *testing.T) {
	tests := []struct {
		name string
		sort []string
		want pagination.Sort
	}{
		{
			name: "omitted",
			want: pagination.Sort{
				Field: "created_at",
				Order: "desc",
			},
		},
		{
			name: "bare field",
			sort: []string{"created_at"},
			want: pagination.Sort{
				Field: "created_at",
				Order: "asc",
			},
		},
		{
			name: "ascending",
			sort: []string{"name:asc"},
			want: pagination.Sort{
				Field: "name",
				Order: "asc",
			},
		},
		{
			name: "descending",
			sort: []string{"created_at:desc"},
			want: pagination.Sort{
				Field: "created_at",
				Order: "desc",
			},
		},
		{
			name: "uppercase ascending",
			sort: []string{"created_at:ASC"},
			want: pagination.Sort{
				Field: "created_at",
				Order: "asc",
			},
		},
		{
			name: "uppercase descending",
			sort: []string{"created_at:DESC"},
			want: pagination.Sort{
				Field: "created_at",
				Order: "desc",
			},
		},
		{
			name: "mixed case ascending",
			sort: []string{"created_at:Asc"},
			want: pagination.Sort{
				Field: "created_at",
				Order: "asc",
			},
		},
		{
			name: "mixed case descending",
			sort: []string{"created_at:dEsC"},
			want: pagination.Sort{
				Field: "created_at",
				Order: "desc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{}
			if tt.sort != nil {
				values["sort"] = tt.sort
			}

			query, err := pagination.NewQuery(values)
			if err != nil {
				t.Fatal(err)
			}

			want := pagination.Query{
				Limit: pagination.DefaultLimit,
				Sort:  tt.want,
			}
			got := query
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("query mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewQuery_InvalidSort(t *testing.T) {
	tests := []struct {
		name string
		sort []string
	}{
		{
			name: "empty",
			sort: []string{""},
		},
		{
			name: "missing field",
			sort: []string{":desc"},
		},
		{
			name: "missing order",
			sort: []string{"name:"},
		},
		{
			name: "unsupported order",
			sort: []string{"name:descending"},
		},
		{
			name: "extra separator",
			sort: []string{"name:asc:desc"},
		},
		{
			name: "whitespace",
			sort: []string{" name:asc"},
		},
		{
			name: "multiple fields",
			sort: []string{"name,created_at"},
		},
		{
			name: "repeated parameter",
			sort: []string{"name:asc", "created_at:desc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{
				"sort": tt.sort,
			}

			_, err := pagination.NewQuery(values)
			isInvalid := errors.Is(err, errs.ErrInvalid)
			if !isInvalid {
				t.Fatalf("error = %v, want invalid input", err)
			}
			validation, ok := errors.AsType[*validatorx.ValidationError](err)
			if !ok {
				t.Fatalf("error = %v, want field failures", err)
			}

			want := "sort"
			for _, field := range validation.Fields {
				got := field.Field
				if diff := cmp.Diff(want, got); diff != "" {
					t.Fatalf("field mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestNewQuery_SortWithCursor(t *testing.T) {
	cursor := pagination.Cursor{
		Version:   1,
		CreatedAt: time.Date(2026, time.October, 31, 12, 0, 0, 0, time.UTC),
		ID:        uuid.UUID{6: 0x70, 8: 0x80},
	}
	token, err := pagination.EncodeCursor(cursor)
	if err != nil {
		t.Fatal(err)
	}

	values := url.Values{
		"sort":   {"created_at:desc"},
		"limit":  {"10"},
		"cursor": {token},
	}

	query, err := pagination.NewQuery(values)
	if err != nil {
		t.Fatal(err)
	}

	want := pagination.Query{
		Limit: 10,
		Sort: pagination.Sort{
			Field: "created_at",
			Order: "desc",
		},
		Cursor: &cursor,
	}
	got := query
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("query mismatch (-want +got):\n%s", diff)
	}
}
