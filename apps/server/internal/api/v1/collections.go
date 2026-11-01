package v1

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
	"uuid"

	"github.com/LucJosin/opid"
	"github.com/go-chi/chi/v5"
	"github.com/lucjosin/qorv.in/internal/domain/collection"
	"github.com/lucjosin/qorv.in/internal/pagination"
	"github.com/lucjosin/qorv.in/internal/queryx"
	"github.com/lucjosin/qorv.in/internal/validatorx"
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
	validator       *validatorx.Validator
}

// NewCollectionHandler registers the collection HTTP handlers.
func NewCollectionHandler(r chi.Router, service collection.Service, collectionCodec, workspaceCodec *opid.Codec) {
	h := &CollectionHandler{
		service:         service,
		collectionCodec: collectionCodec,
		workspaceCodec:  workspaceCodec,
		validator:       validatorx.New(),
	}
	r.Route("/collections", func(r chi.Router) {
		r.Get("/", h.list)
	})
}

type CollectionListParams struct {
	WorkspaceID *uuid.UUID `json:"workspace_id" validate:"omitempty,uuid"`
	Name        *string    `json:"name" validate:"omitnil,min=1,max=255"`
	Category    *string    `json:"category" validate:"omitnil,min=1,max=50"`
	IsPublic    *bool      `json:"is_public"`
	Pinned      *bool      `json:"pinned"`
	PublicStats *bool      `json:"public_stats"`
}

func (h *CollectionHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	values := r.URL.Query()
	query, err := pagination.NewQuery(values)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "Invalid collection parameters", err)
		return
	}

	requestParams, err := h.parseListParams(values)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "Invalid collection parameters", err)
		return
	}

	err = h.validator.Validate(requestParams)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "Invalid collection parameters", err)
		return
	}

	params := collection.ListParams{
		Limit: query.Limit,
		After: query.Cursor,
		Sort:  query.Sort,
		Filters: collection.ListFilters{
			WorkspaceID: requestParams.WorkspaceID,
			Name:        requestParams.Name,
			Category:    requestParams.Category,
			IsPublic:    requestParams.IsPublic,
			Pinned:      requestParams.Pinned,
			PublicStats: requestParams.PublicStats,
		},
	}
	entities, err := h.service.List(ctx, params)
	if err != nil {
		internalError(ctx, w, err)
		return
	}

	page := PaginationResponse{
		PerPage: params.Limit,
	}
	hasNext := len(entities) > params.Limit
	if hasNext {
		entities = entities[:params.Limit]
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

func (h *CollectionHandler) parseListParams(values url.Values) (CollectionListParams, error) {
	var params CollectionListParams

	parser := queryx.New(values)
	workspaceValue, err := parser.String("workspace_id")
	if err != nil {
		return CollectionListParams{}, err
	}
	if workspaceValue != nil {
		workspaceID, err := h.workspaceCodec.Decode(opid.ID(*workspaceValue))
		if err != nil {
			field := validatorx.Field("workspace_id", "invalid workspace ID")
			return CollectionListParams{}, validatorx.NewValidationError(field)
		}
		params.WorkspaceID = &workspaceID
	}

	params.Name, err = parser.String("name")
	if err != nil {
		return CollectionListParams{}, err
	}
	params.Category, err = parser.String("category")
	if err != nil {
		return CollectionListParams{}, err
	}

	params.IsPublic, err = parser.Bool("is_public")
	if err != nil {
		return CollectionListParams{}, err
	}
	params.Pinned, err = parser.Bool("pinned")
	if err != nil {
		return CollectionListParams{}, err
	}
	params.PublicStats, err = parser.Bool("public_stats")
	if err != nil {
		return CollectionListParams{}, err
	}

	return params, nil
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
