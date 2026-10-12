package v1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/LucJosin/opid"
	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/domain/collection"

	v1 "github.com/lucjosin/qorv.in/internal/api/v1"
)

func TestCollectionList_PublicIDs(t *testing.T) {
	entity := collection.Collection{
		ID:          uuid.NewV7(),
		WorkspaceID: uuid.NewV7(),
		Name:        "Bookmarks",
	}
	w := listCollections(t, []collection.Collection{entity})
	if diff := cmp.Diff(http.StatusOK, w.Code); diff != "" {
		t.Fatalf("status mismatch (-want +got):\n%s", diff)
	}

	var response struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(1, len(response.Data)); diff != "" {
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
		if err := json.Unmarshal(data[test.field], &publicID); err != nil {
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
		if diff := cmp.Diff(test.id, id); diff != "" {
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
			if diff := cmp.Diff(http.StatusInternalServerError, w.Code); diff != "" {
				t.Fatalf("status mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectionList_Empty(t *testing.T) {
	w := listCollections(t, nil)
	if diff := cmp.Diff(http.StatusOK, w.Code); diff != "" {
		t.Fatalf("status mismatch (-want +got):\n%s", diff)
	}
	var response v1.APIResponse[[]v1.CollectionResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	want := []v1.CollectionResponse{}
	if diff := cmp.Diff(want, response.Data); diff != "" {
		t.Fatalf("empty collection list mismatch (-want +got):\n%s", diff)
	}
}

type CollectionService struct {
	entities []collection.Collection
}

func (s CollectionService) List(context.Context) ([]collection.Collection, error) {
	return s.entities, nil
}

func listCollections(t *testing.T, entities []collection.Collection) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	service := CollectionService{
		entities: entities,
	}
	collectionCodec, err := opid.NewCodec("col")
	if err != nil {
		t.Fatal(err)
	}
	workspaceCodec, err := opid.NewCodec("ws")
	if err != nil {
		t.Fatal(err)
	}
	v1.NewCollectionHandler(r, service, collectionCodec, workspaceCodec)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/collections/", nil))
	return w
}
