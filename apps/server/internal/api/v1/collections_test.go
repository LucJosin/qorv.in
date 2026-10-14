package v1_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/LucJosin/opid"
	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/domain/collection"
	"github.com/lucjosin/qorv.in/internal/validatorx"

	v1 "github.com/lucjosin/qorv.in/internal/api/v1"
)

func TestCollectionList_PublicIDs(t *testing.T) {
	entity := collection.Collection{
		ID:          uuid.NewV7(),
		WorkspaceID: uuid.NewV7(),
		Name:        "Bookmarks",
	}

	w := listCollections(t, []collection.Collection{entity})
	want := http.StatusOK
	got := w.Code
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("status mismatch (-want +got):\n%s", diff)
	}

	var response struct {
		Data []map[string]json.RawMessage `json:"data"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	want = 1
	got = len(response.Data)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("collection count mismatch (-want +got):\n%s", diff)
	}

	data := response.Data[0]
	_, ok := data["public_id"]
	if ok {
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

	for _, tt := range tests {
		var publicID opid.ID
		err := json.Unmarshal(data[tt.field], &publicID)
		if err != nil {
			t.Fatal(err)
		}

		codec, err := opid.NewCodec(tt.prefix)
		if err != nil {
			t.Fatal(err)
		}

		id, err := codec.Decode(publicID)
		if err != nil {
			t.Fatalf("decoding %s: %v", tt.field, err)
		}

		want := tt.id
		got := id
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("%s mismatch (-want +got):\n%s", tt.field, diff)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := []collection.Collection{
				{
					ID:          tt.id,
					WorkspaceID: tt.workspaceID,
				},
			}

			w := listCollections(t, entities)
			want := http.StatusInternalServerError
			got := w.Code
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("status mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectionList_Empty(t *testing.T) {
	w := listCollections(t, nil)
	wantStatus := http.StatusOK
	gotStatus := w.Code
	if diff := cmp.Diff(wantStatus, gotStatus); diff != "" {
		t.Fatalf("status mismatch (-want +got):\n%s", diff)
	}

	var response v1.APIResponse[[]v1.CollectionResponse]
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	want := []v1.CollectionResponse{}
	got := response.Data
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("empty collection list mismatch (-want +got):\n%s", diff)
	}
}

func TestCollectionList_ValidationErrors(t *testing.T) {
	validator := validatorx.New()
	request := struct {
		Name  string `json:"name" validate:"required"`
		Count int    `json:"count" validate:"min=1"`
		Email string `json:"email" validate:"email"`
	}{}
	validationError := validator.Validate(request)

	tests := []struct {
		name   string
		err    error
		status int
		want   []v1.FieldError
	}{
		{
			name:   "struct validation rules",
			err:    validationError,
			status: http.StatusBadRequest,
			want: []v1.FieldError{
				{
					Field:   "name",
					Key:     "FIELD_REQUIRED",
					Message: "Field name is required",
				},
				{
					Field:   "count",
					Key:     "FIELD_MIN",
					Message: "Field count must be at least 1",
				},
				{
					Field:   "email",
					Key:     "FIELD_INVALID",
					Message: "Field email has an invalid value",
				},
			},
		},
		{
			name: "wrapped multiple fields",
			err: fmt.Errorf("listing collections: %w", &validatorx.ValidationError{
				Fields: []validatorx.FieldError{
					{
						Field:   "workspace_id",
						Tag:     "invalid",
						Message: "invalid workspace",
					},
					{
						Field:  "limit",
						Tag:    "max",
						Params: []string{"100"},
					},
				},
			}),
			status: http.StatusBadRequest,
			want: []v1.FieldError{
				{
					Field:   "workspace_id",
					Key:     "FIELD_INVALID",
					Message: "Field workspace_id invalid workspace",
				},
				{
					Field:   "limit",
					Key:     "FIELD_MAX",
					Message: "Field limit must be at most 100",
				},
			},
		},
		{
			name:   "internal failure",
			err:    errors.New("database failure"),
			status: http.StatusInternalServerError,
		},
		{
			name:   "empty validation error",
			err:    &validatorx.ValidationError{},
			status: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := ValidationCollectionService{
				list: func() ([]collection.Collection, error) {
					return nil, tt.err
				},
			}

			router := chi.NewRouter()
			collectionCodec, err := opid.NewCodec("col")
			if err != nil {
				t.Fatal(err)
			}

			workspaceCodec, err := opid.NewCodec("ws")
			if err != nil {
				t.Fatal(err)
			}
			v1.NewCollectionHandler(router, service, collectionCodec, workspaceCodec)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/collections/", nil))

			want := tt.status
			got := w.Code
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("status mismatch (-want +got):\n%s", diff)
			}

			if tt.status == http.StatusInternalServerError {
				if w.Body.Len() != 0 {
					t.Fatal("internal error exposed a response body")
				}
				return
			}

			var response v1.APIErrorResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			if err != nil {
				t.Fatal(err)
			}

			got = response.Error.StatusCode
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("error status mismatch (-want +got):\n%s", diff)
			}
			wantFields := tt.want
			gotFields := response.Error.Details
			if diff := cmp.Diff(wantFields, gotFields); diff != "" {
				t.Fatalf("fields mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

type ValidationCollectionService struct {
	list func() ([]collection.Collection, error)
}

func (s ValidationCollectionService) List(context.Context) ([]collection.Collection, error) {
	return s.list()
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
