package v1_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestCollectionList_PublicIDs(t *testing.T) {
	entity := collection.Collection{
		ID:          uuid.NewV7(),
		WorkspaceID: uuid.NewV7(),
		Name:        "Bookmarks",
	}
	w := listCollections(t, []collection.Collection{entity})
	got := w.Code
	want := http.StatusOK
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("status mismatch (-want +got):\n%s", diff)
	}

	var response CollectionListRawResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}
	got = len(response.Data)
	want = 1
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("collection count mismatch (-want +got):\n%s", diff)
	}
	data := response.Data[0]
	if _, ok := data["public_id"]; ok {
		t.Fatal("response contains obsolete public_id field")
	}

	tests := []struct {
		field  string
		prefix string
		id     uuid.UUID
	}{
		{
			field:  "id",
			prefix: "col",
			id:     entity.ID,
		},
		{
			field:  "workspace_id",
			prefix: "ws",
			id:     entity.WorkspaceID,
		},
	}
	for _, test := range tests {
		var publicID opid.ID
		err := json.Unmarshal(data[test.field], &publicID)
		if err != nil {
			t.Fatal(err)
		}
		codec, err := opid.NewCodec(test.prefix)
		if err != nil {
			t.Fatal(err)
		}
		id, err := codec.Decode(publicID)
		if err != nil {
			t.Fatalf("decoding %s: %v", test.field, err)
		}
		got := id
		want := test.id
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("%s mismatch (-want +got):\n%s", test.field, diff)
		}
	}
}

func TestCollectionList_RejectsInvalidIDs(t *testing.T) {
	validID := uuid.NewV7()
	tests := []struct {
		name        string
		id          uuid.UUID
		workspaceID uuid.UUID
	}{
		{
			name:        "empty collection",
			workspaceID: validID,
		},
		{
			name: "empty workspace",
			id:   validID,
		},
		{
			name:        "v4 collection",
			id:          uuid.NewV4(),
			workspaceID: validID,
		},
		{
			name:        "v4 workspace",
			id:          validID,
			workspaceID: uuid.NewV4(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entities := []collection.Collection{
				{
					ID:          test.id,
					WorkspaceID: test.workspaceID,
				},
			}
			w := listCollections(t, entities)
			got := w.Code
			want := http.StatusInternalServerError
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("status mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectionList_Empty(t *testing.T) {
	w := listCollections(t, nil)
	{
		got := w.Code
		want := http.StatusOK
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("status mismatch (-want +got):\n%s", diff)
		}
	}
	var response v1.APIResponse[[]v1.CollectionResponse]
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}
	got := response
	want := v1.APIResponse[[]v1.CollectionResponse]{
		Data: []v1.CollectionResponse{},
		Pagination: &pagination.Metadata{
			Limit: pagination.DefaultLimit,
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("empty page mismatch (-want +got):\n%s", diff)
	}
}

func TestCollectionList_PaginationFirstPage(t *testing.T) {
	entities := paginationCollections()
	var params collection.Params
	service := CollectionService{
		list: func(query collection.Params) (pagination.Page[collection.Collection], error) {
			params = query
			return pagination.NewPage(entities, query.Limit), nil
		},
	}
	router := collectionRouter(t, service)

	response := getCollectionPage(t, router, "/collections/?limit=2")
	nextCursor := collectionCursor(t, entities[1])
	got := collectionPageResult(t, response, params)
	want := CollectionPageResult{
		IDs: []uuid.UUID{entities[0].ID, entities[1].ID},
		Pagination: pagination.Metadata{
			Limit:      2,
			HasMore:    true,
			NextCursor: &nextCursor,
		},
		Params: collection.Params{
			Query: pagination.Query{
				Limit: 2,
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("first page mismatch (-want +got):\n%s", diff)
	}
}

func TestCollectionList_PaginationNextPage(t *testing.T) {
	entities := paginationCollections()
	cursor := pagination.Cursor{
		ID:        entities[1].ID,
		CreatedAt: entities[1].CreatedAt,
	}
	var params collection.Params
	service := CollectionService{
		list: func(query collection.Params) (pagination.Page[collection.Collection], error) {
			params = query
			return pagination.NewPage(entities[2:], query.Limit), nil
		},
	}
	router := collectionRouter(t, service)
	token := collectionCursor(t, entities[1])

	response := getCollectionPage(t, router, "/collections/?limit=2&cursor="+url.QueryEscape(token))
	got := collectionPageResult(t, response, params)
	want := CollectionPageResult{
		IDs: []uuid.UUID{entities[2].ID},
		Pagination: pagination.Metadata{
			Limit: 2,
		},
		Params: collection.Params{
			Query: pagination.Query{
				Limit: 2,
				After: &cursor,
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("next page mismatch (-want +got):\n%s", diff)
	}
}

func TestCollectionList_InvalidPagination(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		field   string
		message string
	}{
		{
			name:    "limit",
			query:   "?limit=0",
			field:   "limit",
			message: "must be between 1 and 100",
		},
		{
			name:    "cursor",
			query:   "?cursor=bad",
			field:   "cursor",
			message: "invalid or unsupported cursor",
		},
		{
			name:    "malformed query encoding",
			query:   "?cursor=%ZZ",
			field:   "query",
			message: "invalid query encoding",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := CollectionService{
				list: func(collection.Params) (pagination.Page[collection.Collection], error) {
					t.Fatal("invalid pagination must not call the service")
					return pagination.Page[collection.Collection]{}, nil
				},
			}
			router := collectionRouter(t, service)
			response := getCollectionError(t, router, "/collections/"+test.query, http.StatusBadRequest)
			got := response.Errors
			want := []v1.FieldError{
				{
					Field:   test.field,
					Message: test.message,
					Key:     "invalid",
				},
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("fields mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectionList_InternalError(t *testing.T) {
	service := CollectionService{
		list: func(collection.Params) (pagination.Page[collection.Collection], error) {
			return pagination.Page[collection.Collection]{}, errors.New("secret database credentials")
		},
	}
	router := collectionRouter(t, service)
	response := getCollectionError(t, router, "/collections/", http.StatusInternalServerError)
	got := response.Message
	want := "An internal server error occurred."
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("safe message mismatch (-want +got):\n%s", diff)
	}
}

type CollectionService struct {
	entities []collection.Collection
	list     func(collection.Params) (pagination.Page[collection.Collection], error)
}

type CollectionListRawResponse struct {
	Data []map[string]json.RawMessage `json:"data"`
}

func (s CollectionService) List(_ context.Context, request collection.Params) (pagination.Page[collection.Collection], error) {
	if s.list != nil {
		return s.list(request)
	}
	return pagination.NewPage(s.entities, request.Limit), nil
}

func listCollections(t *testing.T, entities []collection.Collection) *httptest.ResponseRecorder {
	t.Helper()
	service := CollectionService{
		entities: entities,
	}
	r := collectionRouter(t, service)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/collections/", nil))
	return w
}

func collectionRouter(t *testing.T, service collection.Service) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	collectionCodec, err := opid.NewCodec("col")
	if err != nil {
		t.Fatal(err)
	}
	workspaceCodec, err := opid.NewCodec("ws")
	if err != nil {
		t.Fatal(err)
	}
	v1.NewCollectionHandler(r, service, collectionCodec, workspaceCodec)
	return r
}

func getCollectionPage(t *testing.T, router http.Handler, path string) v1.APIResponse[[]v1.CollectionResponse] {
	t.Helper()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	got := w.Code
	want := http.StatusOK
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("status mismatch (-want +got):\n%s", diff)
	}
	var response v1.APIResponse[[]v1.CollectionResponse]
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.Pagination == nil {
		t.Fatal("missing pagination metadata")
	}
	return response
}

func getCollectionError(t *testing.T, router http.Handler, path string, status int) v1.ErrorResponse {
	t.Helper()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	got := w.Code
	want := status
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("status mismatch (-want +got):\n%s", diff)
	}
	var response v1.APIErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}
	got = response.Error.StatusCode
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("error status mismatch (-want +got):\n%s", diff)
	}
	{
		got := response.Error.Path
		want := "/collections/"
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("path mismatch (-want +got):\n%s", diff)
		}
	}
	_, err = time.Parse(time.RFC3339Nano, response.Error.Timestamp)
	if err != nil {
		t.Fatal(err)
	}
	return response.Error
}

type CollectionPageResult struct {
	IDs        []uuid.UUID
	Pagination pagination.Metadata
	Params     collection.Params
}

func paginationCollections() []collection.Collection {
	createdAt := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	workspaceID := uuid.NewV7()
	ids := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	entities := make([]collection.Collection, 0, len(ids))
	for i := len(ids) - 1; i >= 0; i-- {
		entity := collection.Collection{
			ID:          ids[i],
			WorkspaceID: workspaceID,
			CreatedAt:   createdAt,
		}
		entities = append(entities, entity)
	}
	return entities
}

func collectionCursor(t *testing.T, entity collection.Collection) string {
	t.Helper()
	codec, err := opid.NewCodec("col")
	if err != nil {
		t.Fatal(err)
	}
	cursor := pagination.Cursor{
		CreatedAt: entity.CreatedAt,
		ID:        entity.ID,
	}
	var params collection.Params
	token, err := pagination.Encode(cursor, params, codec)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func collectionPageResult(t *testing.T, response v1.APIResponse[[]v1.CollectionResponse], params collection.Params) CollectionPageResult {
	t.Helper()
	codec, err := opid.NewCodec("col")
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]uuid.UUID, 0, len(response.Data))
	for _, item := range response.Data {
		id, err := codec.Decode(item.ID)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	result := CollectionPageResult{
		IDs:        ids,
		Pagination: *response.Pagination,
		Params:     params,
	}
	return result
}
