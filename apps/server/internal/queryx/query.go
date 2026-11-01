// Package queryx reads optional query parameters and converts scalar values.
package queryx

import (
	"net/url"
	"strconv"

	"github.com/lucjosin/qorv.in/internal/validatorx"
)

// Parser reads query values without modifying them.
// Its zero value behaves like a request with no query parameters.
type Parser struct {
	values url.Values
}

func New(values url.Values) Parser {
	parser := Parser{
		values: values,
	}
	return parser
}

// String returns nil when the parameter is absent and rejects repeated values.
// An explicit empty value returns a pointer to "" for the caller to validate.
// Whitespace is preserved; defaults and domain rules belong to the caller.
func (p Parser) String(name string) (*string, error) {
	values := p.values[name]
	if len(values) == 0 {
		return nil, nil
	}
	if len(values) > 1 {
		field := validatorx.Field(name, "must be provided once")
		return nil, validatorx.NewValidationError(field)
	}

	value := values[0]
	return &value, nil
}

// Bool returns nil when the parameter is absent and preserves explicit false.
// It accepts the forms supported by strconv.ParseBool. Empty, malformed, and
// repeated values return field validation errors.
func (p Parser) Bool(name string) (*bool, error) {
	value, err := p.String(name)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}

	parsed, err := strconv.ParseBool(*value)
	if err != nil {
		field := validatorx.Field(name, "must be a boolean")
		return nil, validatorx.NewValidationError(field)
	}

	return &parsed, nil
}
