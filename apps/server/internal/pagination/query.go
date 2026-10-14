package pagination

import (
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

const (
	// DefaultLimit is the page size when no limit is provided.
	DefaultLimit = 20
	// MaxLimit bounds the number of records returned per page.
	MaxLimit = 100
)

type Query struct {
	Limit int
	After *Cursor
}

func (r Query) Validate() error {
	if r.Limit < 1 || r.Limit > MaxLimit {
		return validatorx.NewValidationError(validatorx.Field("limit", "must be between 1 and 100"))
	}
	if r.After != nil && (r.After.CreatedAt.IsZero() || r.After.CreatedAt.Year() < 1970 || r.After.CreatedAt.Year() > 9999 || r.After.ID[6]>>4 != 7 || r.After.ID[8]>>6 != 2) {
		return validatorx.NewValidationError(validatorx.Field("cursor", "invalid cursor position"))
	}
	return nil
}
