package pagination

import (
	"net/url"
	"strconv"

	"github.com/LucJosin/opid"
	"github.com/lucjosin/qorv.in/internal/errs"
)

// ParseQuery accepts only limit and cursor and validates their encoding.
func ParseQuery[F any](rawQuery string, filters F, codec *opid.Codec) (Query, error) {
	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		return Query{}, &errs.ValidationError{Field: "query", Message: "invalid query encoding"}
	}
	return Parse(query, filters, codec)
}

// Parse validates pagination parameters and decodes a continuation position.
func Parse[F any](query url.Values, filters F, codec *opid.Codec) (Query, error) {
	r := Query{Limit: DefaultLimit}
	for field, values := range query {
		if field != "limit" && field != "cursor" {
			return Query{}, &errs.ValidationError{Field: field, Message: "unsupported query parameter"}
		}
		if len(values) != 1 || values[0] == "" {
			return Query{}, &errs.ValidationError{Field: field, Message: "must have exactly one non-empty value"}
		}
	}
	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			return Query{}, &errs.ValidationError{Field: "limit", Message: "must be an integer between 1 and 100"}
		}
		r.Limit = limit
	}
	if value := query.Get("cursor"); value != "" {
		position, err := decode(value, filters, codec)
		if err != nil {
			return Query{}, &errs.ValidationError{Field: "cursor", Message: "invalid or unsupported cursor"}
		}
		r.After = &position
	}
	err := r.Validate()
	if err != nil {
		return Query{}, err
	}
	return r, nil
}
