package pagination

import (
	"net/url"
	"strconv"
	"uuid"

	"github.com/lucjosin/qorv.in/internal/validatorx"
)

type Query struct {
	Limit  int
	Sort   Sort
	Cursor *Cursor
}

func (r Query) Validate() error {
	var errs []validatorx.FieldError

	if r.Limit < 1 || r.Limit > MaxLimit {
		errs = append(errs, validatorx.Field("limit", "must be between 1 and 100"))
	}
	if r.Cursor != nil {
		zeroTime := r.Cursor.CreatedAt.IsZero()
		if zeroTime || !isUUIDv7(r.Cursor.ID) {
			errs = append(errs, validatorx.Field("cursor", "invalid cursor position"))
		}
	}

	return validatorx.NewValidationError(errs...)
}

func NewQuery(values url.Values) (Query, error) {
	cursorStr := values.Get("cursor")
	limitStr := values.Get("limit")

	sort, err := parseSort(values["sort"])
	if err != nil {
		return Query{}, err
	}

	limit := DefaultLimit
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			return Query{}, validatorx.NewValidationError(
				validatorx.Field("limit", "must be an integer between 1 and 100"),
			)
		}
		limit = l
	}

	query := Query{
		Limit: limit,
		Sort:  sort,
	}
	if cursorStr != "" {
		cursor, err := DecodeCursor(cursorStr)
		if err != nil {
			return Query{}, validatorx.NewValidationError(
				validatorx.Field("cursor", "invalid cursor position"),
			)
		}
		query.Cursor = &cursor
	}

	err = query.Validate()
	if err != nil {
		return Query{}, err
	}

	return query, nil
}

func isUUIDv7(id uuid.UUID) bool {
	return id[6]>>4 == 7 && id[8]&0xC0 == 0x80
}
