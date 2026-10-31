package v1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := paginationCollectionService{}
			router := chi.NewRouter()
			v1.NewCollectionHandler(router, service, nil, nil)
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

type paginationCollectionService struct {
	entities []collection.Collection
}

func (s paginationCollectionService) List(context.Context, collection.ListParams) ([]collection.Collection, error) {
	return s.entities, nil
}
