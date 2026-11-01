package v1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/LucJosin/opid"
	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/domain/collection"
	"github.com/lucjosin/qorv.in/internal/pagination"

	v1 "github.com/lucjosin/qorv.in/internal/api/v1"
)

func TestCollectionList_Pagination(t *testing.T) {
	first := collection.Collection{
		ID:          uuid.NewV7(),
		WorkspaceID: uuid.NewV7(),
		Name:        "First",
		CreatedAt:   time.Date(2026, time.October, 31, 12, 0, 0, 0, time.UTC),
	}
	second := collection.Collection{
		ID:          uuid.NewV7(),
		WorkspaceID: first.WorkspaceID,
		Name:        "Second",
		CreatedAt:   first.CreatedAt.Add(-time.Hour),
	}
	collectionCodec, err := opid.NewCodec("col")
	if err != nil {
		t.Fatal(err)
	}
	workspaceCodec, err := opid.NewCodec("ws")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		entities []collection.Collection
		want     []string
		hasNext  bool
	}{
		{
			name: "nil list",
			want: []string{},
		},
		{
			name:     "empty list",
			entities: []collection.Collection{},
			want:     []string{},
		},
		{
			name:     "partial page",
			entities: []collection.Collection{first},
			want:     []string{"First"},
		},
		{
			name:     "full final page",
			entities: []collection.Collection{first, second},
			want:     []string{"First", "Second"},
		},
		{
			name:     "lookahead is trimmed before response conversion",
			entities: []collection.Collection{first, second, {}},
			want:     []string{"First", "Second"},
			hasNext:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := paginationCollectionService{
				entities: tt.entities,
			}
			router := chi.NewRouter()
			v1.NewCollectionHandler(router, service, collectionCodec, workspaceCodec)
			request := httptest.NewRequest(http.MethodGet, "/collections/?limit=2", nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			wantStatus := http.StatusOK
			gotStatus := recorder.Code
			if diff := cmp.Diff(wantStatus, gotStatus); diff != "" {
				t.Fatalf("status mismatch (-want +got):\n%s", diff)
			}
			var response v1.APIResponse[[]v1.CollectionResponse]
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			if err != nil {
				t.Fatal(err)
			}

			if response.Data == nil {
				t.Fatal("data must be an array")
			}
			want := tt.want
			got := make([]string, 0, len(response.Data))
			for _, entity := range response.Data {
				got = append(got, entity.Name)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("data mismatch (-want +got):\n%s", diff)
			}
			if response.Pagination == nil {
				t.Fatal("missing pagination metadata")
			}
			wantPage := v1.PaginationResponse{
				PerPage: 2,
			}
			gotPage := *response.Pagination
			gotPage.NextCursor = ""
			if diff := cmp.Diff(wantPage, gotPage); diff != "" {
				t.Fatalf("pagination mismatch (-want +got):\n%s", diff)
			}
			if !tt.hasNext {
				if response.Pagination.NextCursor != "" {
					t.Fatal("final page must not include a next cursor")
				}
				return
			}

			cursor, err := pagination.DecodeCursor(response.Pagination.NextCursor)
			if err != nil {
				t.Fatal(err)
			}

			wantCursor := pagination.Cursor{
				Version:   1,
				CreatedAt: second.CreatedAt,
				ID:        second.ID,
			}
			gotCursor := cursor
			if diff := cmp.Diff(wantCursor, gotCursor); diff != "" {
				t.Fatalf("cursor mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectionList_InvalidQuery(t *testing.T) {
	workspaceCodec, err := opid.NewCodec("ws")
	if err != nil {
		t.Fatal(err)
	}
	collectionCodec, err := opid.NewCodec("col")
	if err != nil {
		t.Fatal(err)
	}
	collectionID, err := collectionCodec.Encode(uuid.NewV7())
	if err != nil {
		t.Fatal(err)
	}
	cursor := pagination.Cursor{
		Version:   2,
		CreatedAt: time.Date(2026, time.October, 31, 12, 0, 0, 0, time.UTC),
		ID:        uuid.NewV7(),
	}
	cursorToken, err := pagination.EncodeCursor(cursor)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		query string
		field string
	}{
		{
			name:  "malformed limit",
			query: "limit=abc",
			field: "limit",
		},
		{
			name:  "out of range limit",
			query: "limit=101",
			field: "limit",
		},
		{
			name:  "malformed cursor",
			query: "cursor=invalid",
			field: "cursor",
		},
		{
			name:  "malformed sort",
			query: "sort=created_at:invalid",
			field: "sort",
		},
		{
			name:  "unsupported cursor version",
			query: "cursor=" + cursorToken,
			field: "cursor",
		},
		{
			name:  "empty workspace",
			query: "workspace_id=",
			field: "workspace_id",
		},
		{
			name:  "malformed workspace",
			query: "workspace_id=invalid",
			field: "workspace_id",
		},
		{
			name:  "raw workspace UUID",
			query: "workspace_id=" + uuid.NewV7().String(),
			field: "workspace_id",
		},
		{
			name:  "wrong resource prefix",
			query: "workspace_id=" + string(collectionID),
			field: "workspace_id",
		},
		{
			name:  "repeated workspace",
			query: "workspace_id=one&workspace_id=two",
			field: "workspace_id",
		},
		{
			name:  "empty name",
			query: "name=",
			field: "name",
		},
		{
			name:  "long name",
			query: "name=" + strings.Repeat("a", 256),
			field: "name",
		},
		{
			name:  "repeated name",
			query: "name=First&name=Second",
			field: "name",
		},
		{
			name:  "empty category",
			query: "category=",
			field: "category",
		},
		{
			name:  "long category",
			query: "category=" + strings.Repeat("a", 51),
			field: "category",
		},
		{
			name:  "repeated category",
			query: "category=Design&category=Tools",
			field: "category",
		},
		{
			name:  "invalid public flag",
			query: "is_public=invalid",
			field: "is_public",
		},
		{
			name:  "empty public flag",
			query: "is_public=",
			field: "is_public",
		},
		{
			name:  "repeated public flag",
			query: "is_public=true&is_public=false",
			field: "is_public",
		},
		{
			name:  "invalid pinned flag",
			query: "pinned=invalid",
			field: "pinned",
		},
		{
			name:  "empty pinned flag",
			query: "pinned=",
			field: "pinned",
		},
		{
			name:  "repeated pinned flag",
			query: "pinned=true&pinned=false",
			field: "pinned",
		},
		{
			name:  "invalid public stats flag",
			query: "public_stats=invalid",
			field: "public_stats",
		},
		{
			name:  "empty public stats flag",
			query: "public_stats=",
			field: "public_stats",
		},
		{
			name:  "repeated public stats flag",
			query: "public_stats=true&public_stats=false",
			field: "public_stats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := paginationCollectionService{
				list: func(context.Context, collection.ListParams) ([]collection.Collection, error) {
					t.Fatal("service called for invalid parameters")
					return nil, nil
				},
			}
			router := chi.NewRouter()
			v1.NewCollectionHandler(router, service, nil, workspaceCodec)
			request := httptest.NewRequest(http.MethodGet, "/collections/?"+tt.query, nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			wantStatus := http.StatusBadRequest
			gotStatus := recorder.Code
			if diff := cmp.Diff(wantStatus, gotStatus); diff != "" {
				t.Fatalf("status mismatch (-want +got):\n%s", diff)
			}
			var response v1.APIErrorResponse
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			if err != nil {
				t.Fatal(err)
			}

			want := []string{tt.field}
			got := make([]string, 0, len(response.Error.Details))
			for _, detail := range response.Error.Details {
				got = append(got, detail.Field)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("fields mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectionList_WorkspaceFilter(t *testing.T) {
	workspaceCodec, err := opid.NewCodec("ws")
	if err != nil {
		t.Fatal(err)
	}
	workspaceID := uuid.NewV7()
	publicID, err := workspaceCodec.Encode(workspaceID)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		query string
		want  *uuid.UUID
	}{
		{
			name: "omitted",
		},
		{
			name:  "workspace OpID",
			query: "workspace_id=" + string(publicID),
			want:  &workspaceID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var got *uuid.UUID
			service := paginationCollectionService{
				list: func(_ context.Context, params collection.ListParams) ([]collection.Collection, error) {
					called = true
					got = params.Filters.WorkspaceID
					return nil, nil
				},
			}
			router := chi.NewRouter()
			v1.NewCollectionHandler(router, service, nil, workspaceCodec)
			request := httptest.NewRequest(http.MethodGet, "/collections/?"+tt.query, nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			wantStatus := http.StatusOK
			gotStatus := recorder.Code
			if diff := cmp.Diff(wantStatus, gotStatus); diff != "" {
				t.Fatalf("status mismatch (-want +got):\n%s", diff)
			}
			if !called {
				t.Fatal("service was not called")
			}
			want := tt.want
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("workspace mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectionList_Filters(t *testing.T) {
	name := "Demo collection"
	literalName := "50%_done!"
	category := "Research Notes"
	trueValue := true
	falseValue := false
	tests := []struct {
		name  string
		query string
		want  collection.ListFilters
	}{
		{
			name: "omitted",
		},
		{
			name:  "name",
			query: "name=Demo+collection",
			want: collection.ListFilters{
				Name: &name,
			},
		},
		{
			name:  "name preserves literal wildcard characters",
			query: "name=50%25_done%21",
			want: collection.ListFilters{
				Name: &literalName,
			},
		},
		{
			name:  "category",
			query: "category=Research+Notes",
			want: collection.ListFilters{
				Category: &category,
			},
		},
		{
			name:  "public only",
			query: "is_public=true",
			want: collection.ListFilters{
				IsPublic: &trueValue,
			},
		},
		{
			name:  "private only",
			query: "is_public=false",
			want: collection.ListFilters{
				IsPublic: &falseValue,
			},
		},
		{
			name:  "pinned only",
			query: "pinned=true",
			want: collection.ListFilters{
				Pinned: &trueValue,
			},
		},
		{
			name:  "unpinned only",
			query: "pinned=false",
			want: collection.ListFilters{
				Pinned: &falseValue,
			},
		},
		{
			name:  "public stats enabled",
			query: "public_stats=true",
			want: collection.ListFilters{
				PublicStats: &trueValue,
			},
		},
		{
			name:  "public stats disabled",
			query: "public_stats=false",
			want: collection.ListFilters{
				PublicStats: &falseValue,
			},
		},
		{
			name:  "combined filters",
			query: "name=Demo+collection&category=Research+Notes&is_public=false&pinned=true&public_stats=false",
			want: collection.ListFilters{
				Name:        &name,
				Category:    &category,
				IsPublic:    &falseValue,
				Pinned:      &trueValue,
				PublicStats: &falseValue,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var got collection.ListFilters
			service := paginationCollectionService{
				list: func(_ context.Context, params collection.ListParams) ([]collection.Collection, error) {
					called = true
					got = params.Filters
					return nil, nil
				},
			}
			router := chi.NewRouter()
			v1.NewCollectionHandler(router, service, nil, nil)
			request := httptest.NewRequest(http.MethodGet, "/collections/?"+tt.query, nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			wantStatus := http.StatusOK
			gotStatus := recorder.Code
			if diff := cmp.Diff(wantStatus, gotStatus); diff != "" {
				t.Fatalf("status mismatch (-want +got):\n%s", diff)
			}
			if !called {
				t.Fatal("service was not called")
			}
			want := tt.want
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("filters mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type paginationCollectionService struct {
	entities []collection.Collection
	list     func(context.Context, collection.ListParams) ([]collection.Collection, error)
}

func (s paginationCollectionService) List(ctx context.Context, params collection.ListParams) ([]collection.Collection, error) {
	if s.list != nil {
		return s.list(ctx, params)
	}
	return s.entities, nil
}
