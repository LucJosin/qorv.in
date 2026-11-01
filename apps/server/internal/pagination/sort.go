package pagination

import (
	"strings"
	"unicode"

	"github.com/lucjosin/qorv.in/internal/validatorx"
)

type Sort struct {
	Field string
	Order string
}

// parseSort accepts a single field with an optional :asc or :desc suffix.
// Sort directions are case-insensitive and normalized to lowercase.
// Allowed fields are validated by the domain using the query.
func parseSort(values []string) (Sort, error) {
	if len(values) > 1 {
		return Sort{}, validatorx.NewValidationError(
			validatorx.Field("sort", "must be provided once"),
		)
	}

	value := DefaultSort
	if len(values) == 1 {
		value = values[0]
	}

	field, order, hasOrder := strings.Cut(value, ":")
	if !hasOrder {
		order = "asc"
	}
	order = strings.ToLower(order)

	invalidField := strings.ContainsAny(field, ":,;+-") || strings.ContainsFunc(field, unicode.IsSpace)
	if field == "" || invalidField || (order != "asc" && order != "desc") {
		return Sort{}, validatorx.NewValidationError(
			validatorx.Field("sort", "must be a field optionally followed by :asc or :desc"),
		)
	}

	sort := Sort{
		Field: field,
		Order: order,
	}
	return sort, nil
}
