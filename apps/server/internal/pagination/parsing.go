package pagination

import (
	"net/url"
	"strconv"

	"github.com/LucJosin/opid"
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

// ParseQuery accepts only limit and cursor and validates their encoding.
func ParseQuery[F any](rawQuery string, filters F, codec *opid.Codec) (Query, error) {
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return Query{}, validatorx.NewValidationError(validatorx.Field("query", "invalid query encoding"))
	}
	return Parse(query, filters, codec)
}

// Parse validates pagination parameters and decodes a continuation position.
func Parse[F any](query url.Values, filters F, codec *opid.Codec) (Query, error) {
	r := Query{Limit: DefaultLimit}
	for field, values := range query {
		if field != "limit" && field != "cursor" {
			return Query{}, validatorx.NewValidationError(validatorx.Field(field, "unsupported query parameter"))
		}
		if len(values) != 1 || values[0] == "" {
			return Query{}, validatorx.NewValidationError(validatorx.Field(field, "must have exactly one non-empty value"))
		}
	}
	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			return Query{}, validatorx.NewValidationError(validatorx.Field("limit", "must be an integer between 1 and 100"))
		}
		r.Limit = limit
	}
	if value := query.Get("cursor"); value != "" {
		position, err := decode(value, filters, codec)
		if err != nil {
			return Query{}, validatorx.NewValidationError(validatorx.Field("cursor", "invalid or unsupported cursor"))
		}
		r.After = &position
	}
	err := r.Validate()
	if err != nil {
		return Query{}, err
	}
	return r, nil
}
