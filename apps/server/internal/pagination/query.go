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
	var errs []validatorx.FieldError

	if r.Limit < 1 || r.Limit > MaxLimit {
		errs = append(errs, validatorx.Field("limit", "must be between 1 and 100"))
	}
	if r.After != nil {
		zeroTime := r.After.CreatedAt.IsZero()
		year := r.After.CreatedAt.Year()
		validID := r.After.ID[6]>>4 == 7 && r.After.ID[8]>>6 == 2
		if zeroTime || year < 1970 || year > 9999 || !validID {
			errs = append(errs, validatorx.Field("cursor", "invalid cursor position"))
		}
	}

	return validatorx.NewValidationError(errs...)
}
