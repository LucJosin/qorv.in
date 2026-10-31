package v1

import (
	"fmt"
	"net/http"
	"time"

	"github.com/LucJosin/opid"
	"github.com/go-chi/chi/v5"
	"github.com/lucjosin/qorv.in/internal/domain/collection"
	"github.com/lucjosin/qorv.in/internal/pagination"
)

type CollectionResponse struct {
	ID          opid.ID    `json:"id"`
	WorkspaceID opid.ID    `json:"workspace_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitzero"`
	Category    *string    `json:"category,omitzero"`
	IsPublic    bool       `json:"is_public"`
	Color       *string    `json:"color,omitzero"`
	Icon        *string    `json:"icon,omitzero"`
	PublicStats bool       `json:"public_stats"`
	Pinned      bool       `json:"pinned"`
	DeletedAt   *time.Time `json:"deleted_at,omitzero"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CollectionHandler struct {
	service         collection.Service
	collectionCodec *opid.Codec
	workspaceCodec  *opid.Codec
}

// NewCollectionHandler registers the collection HTTP handlers.
func NewCollectionHandler(r chi.Router, service collection.Service, collectionCodec, workspaceCodec *opid.Codec) {
	h := &CollectionHandler{
		service:         service,
		collectionCodec: collectionCodec,
		workspaceCodec:  workspaceCodec,
	}
	r.Route("/collections", func(r chi.Router) {
		r.Get("/", h.list)
	})
}

func (h *CollectionHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query, err := pagination.NewQuery(r.URL.Query())
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "Invalid collection parameters", err)
		return
	}

	params := collection.ListParams{
		Limit:   query.Limit,
		After:   query.Cursor,
		Sort:    query.Sort,
		Filters: collection.ListFilters{},
	}
	err = params.Validate()
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "Invalid collection parameters", err)
		return
	}

	entities, err := h.service.List(ctx, params)
	if err != nil {
		internalError(ctx, w, err)
		return
	}

	page := PaginationResponse{
		PerPage: query.Limit,
	}
	hasNext := len(entities) > query.Limit
	if hasNext {
		entities = entities[:query.Limit]
		last := entities[len(entities)-1]
		cursor := pagination.Cursor{
			Version:   1,
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		}
		nextCursor, err := pagination.EncodeCursor(cursor)
		if err != nil {
			internalError(ctx, w, err)
			return
		}
		page.NextCursor = nextCursor
	}

	items := make([]CollectionResponse, 0, len(entities))
	for _, entity := range entities {
		entity, err := h.toResponse(entity)
		if err != nil {
			internalError(ctx, w, err)
			return
		}
		items = append(items, entity)
	}

	response := APIResponse[[]CollectionResponse]{
		Data:       items,
		Pagination: &page,
	}
	writeResponse(ctx, w, http.StatusOK, response)
}

func (h *CollectionHandler) toResponse(e collection.Collection) (CollectionResponse, error) {
	id, err := h.collectionCodec.Encode(e.ID)
	if err != nil {
		return CollectionResponse{}, fmt.Errorf("encoding collection ID: %w", err)
	}
	workspaceID, err := h.workspaceCodec.Encode(e.WorkspaceID)
	if err != nil {
		return CollectionResponse{}, fmt.Errorf("encoding workspace ID: %w", err)
	}

	res := CollectionResponse{
		ID:          id,
		WorkspaceID: workspaceID,
		Name:        e.Name,
		Category:    e.Category,
		IsPublic:    e.IsPublic,
		Color:       e.Color,
		Icon:        e.Icon,
		PublicStats: e.PublicStats,
		Pinned:      e.Pinned,
		Description: e.Description,
		DeletedAt:   e.DeletedAt,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
	return res, nil
}
