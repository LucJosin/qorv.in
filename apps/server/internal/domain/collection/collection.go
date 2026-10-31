package collection

import (
	"slices"
	"time"
	"uuid"

	"github.com/lucjosin/qorv.in/internal/pagination"
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

var ValidSortFields = []string{
	"created_at",
}

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
	WorkspaceID uuid.UUID
}

type ListParams struct {
	Limit   int
	After   *pagination.Cursor
	Sort    pagination.Sort
	Filters ListFilters
}

func (p ListParams) Validate() error {
	var errs []validatorx.FieldError

	if !slices.Contains(ValidSortFields, p.Sort.Field) {
		errs = append(errs, validatorx.Field("sort", "Invalid sort field"))
	}

	return validatorx.NewValidationError(errs...)
}
