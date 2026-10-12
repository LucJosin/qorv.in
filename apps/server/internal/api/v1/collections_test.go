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
	v1 "github.com/lucjosin/qorv.in/internal/api/v1"
	"github.com/lucjosin/qorv.in/internal/domain/collection"
)

type collectionService struct {
	entities []collection.Collection
}

func (s collectionService) List(context.Context) ([]collection.Collection, error) {
	return s.entities, nil
}

func listCollections(t *testing.T, entities []collection.Collection) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	if err := v1.NewCollectionHandler(r, collectionService{entities: entities}); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/collections/", nil))
	return w
}

func TestCollectionListPublicIDs(t *testing.T) {
	entity := collection.Collection{
		ID:          uuid.NewV7(),
		WorkspaceID: uuid.NewV7(),
		Name:        "Bookmarks",
	}
	w := listCollections(t, []collection.Collection{entity})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var response struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data) != 1 {
		t.Fatalf("collection count = %d, want 1", len(response.Data))
	}
	data := response.Data[0]
	if _, ok := data["public_id"]; ok {
		t.Fatal("response contains obsolete public_id field")
	}

	for _, test := range []struct {
		field  string
		prefix string
		id     uuid.UUID
	}{
		{field: "id", prefix: "col", id: entity.ID},
		{field: "workspace_id", prefix: "ws", id: entity.WorkspaceID},
	} {
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
		if id != test.id {
			t.Fatalf("%s decoded to %s, want %s", test.field, id, test.id)
		}
	}
}

func TestCollectionListRejectsInvalidIDs(t *testing.T) {
	validID := uuid.NewV7()
	for _, test := range []struct {
		name        string
		id          uuid.UUID
		workspaceID uuid.UUID
	}{
		{name: "empty collection", workspaceID: validID},
		{name: "empty workspace", id: validID},
		{name: "v4 collection", id: uuid.NewV4(), workspaceID: validID},
		{name: "v4 workspace", id: validID, workspaceID: uuid.NewV4()},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := listCollections(t, []collection.Collection{{ID: test.id, WorkspaceID: test.workspaceID}})
			if w.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
			}
		})
	}
}

func TestCollectionListEmpty(t *testing.T) {
	w := listCollections(t, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var response v1.APIResponse[[]v1.CollectionResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data == nil || len(response.Data) != 0 {
		t.Fatalf("empty collection list = %#v, want an empty array", response.Data)
	}
}
