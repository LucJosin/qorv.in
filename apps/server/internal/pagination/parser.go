package pagination

import (
	"errors"
	"net/url"
	"strconv"

	"github.com/LucJosin/opid"
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

// ParseQuery validates query encoding and parses pagination parameters. Other parameters are ignored.
func ParseQuery[F any](rawQuery string, filters F, codec *opid.Codec) (Query, error) {
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return Query{}, validatorx.NewValidationError(validatorx.Field("query", "invalid query encoding"))
	}
	return Parse(query, filters, codec)
}

// Parse validates pagination parameters and decodes a continuation position.
func Parse[F any](query url.Values, filters F, codec *opid.Codec) (Query, error) {
	r := Query{
		Limit: DefaultLimit,
	}
	for _, field := range []string{"limit", "cursor"} {
		values, ok := query[field]
		if ok && (len(values) != 1 || values[0] == "") {
			return Query{}, validatorx.NewValidationError(validatorx.Field(field, "must have exactly one non-empty value"))
		}
	}

	var errs []validatorx.FieldError
	limitStr := query.Get("limit")
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			errs = append(errs, validatorx.Field("limit", "must be an integer between 1 and 100"))
		} else {
			r.Limit = limit
		}
	}

	cursorStr := query.Get("cursor")
	if cursorStr != "" {
		cursor, err := decode(cursorStr, filters, codec)
		if err != nil {
			errs = append(errs, validatorx.Field("cursor", "invalid or unsupported cursor"))
		} else {
			r.After = &cursor
		}
	}

	err := r.Validate()
	if err != nil {
		validation, ok := errors.AsType[*validatorx.ValidationError](err)
		if !ok {
			return Query{}, err
		}
		errs = append(errs, validation.Fields...)
	}
	if len(errs) > 0 {
		return Query{}, validatorx.NewValidationError(errs...)
	}

	return r, nil
}
