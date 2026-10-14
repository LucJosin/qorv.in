package validatorx

import "github.com/lucjosin/qorv.in/internal/errs"

// FieldError describes a validation rule failure or a manually supplied message.
type FieldError struct {
	Field   string
	Tag     string
	Message string
	Params  []string
}

// ValidationError contains field failures and wraps errs.ErrInvalid.
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string {
	return errs.ErrInvalid.Error()
}

func (e *ValidationError) Unwrap() error {
	return errs.ErrInvalid
}

// NewValidationError returns nil when there are no field failures.
func NewValidationError(fields ...FieldError) error {
	if len(fields) == 0 {
		return nil
	}
	return &ValidationError{
		Fields: fields,
	}
}

// Field constructs a manually checked field failure.
func Field(field, message string, params ...string) FieldError {
	return FieldError{
		Field:   field,
		Tag:     "invalid",
		Message: message,
		Params:  params,
	}
}
