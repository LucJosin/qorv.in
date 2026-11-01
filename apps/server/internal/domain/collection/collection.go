package collection

import (
	"time"
	"uuid"

	"github.com/lucjosin/qorv.in/internal/pagination"
)

type Collection struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Description *string
	Category    *string
	IsPublic    bool
	Color       *string
	Icon        *string
	PublicStats bool
	Pinned      bool
	DeletedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ListFilters struct {
	WorkspaceID *uuid.UUID
	Name        *string
	Category    *string
	IsPublic    *bool
	Pinned      *bool
	PublicStats *bool
}

type ListParams struct {
	Limit   int
	After   *pagination.Cursor
	Sort    pagination.Sort
	Filters ListFilters
}
