// Package validatorx adapts struct validation to application errors.
package validatorx

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator caches struct metadata and can be shared between goroutines.
// Construct it once and reuse it. The zero value is not initialized.
type Validator struct {
	validator *validator.Validate
}

// New configures JSON field names and required validation for nested structs.
//
// Use validate:"-" to skip validation. The json:"-" tag only controls serialization.
func New() *Validator {
	v := &Validator{
		validator: validator.New(validator.WithRequiredStructEnabled()),
	}
	v.WithJSONTag()
	return v
}

// WithJSONTag uses JSON tag names in field validation errors.
// Configure it before validating values or sharing the validator between goroutines.
//
// Use validate:"-" to skip validation. The json:"-" tag does not skip validation.
func (v *Validator) WithJSONTag() {
	v.validator.RegisterTagNameFunc(func(field reflect.StructField) string {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})
}

// Validate checks a struct or struct pointer. Invalid input types remain internal
// errors. Only field validation failures wrap errs.ErrInvalid.
func (v *Validator) Validate(value any) error {
	err := v.validator.Struct(value)
	if err == nil {
		return nil
	}

	var failures validator.ValidationErrors
	isValidationError := errors.As(err, &failures)
	if !isValidationError {
		return err
	}

	fields := make([]FieldError, 0, len(failures))
	for _, failure := range failures {
		var params []string
		param := failure.Param()
		if param != "" {
			params = []string{param}
		}
		fields = append(fields, FieldError{
			Field:  failure.Field(),
			Tag:    failure.Tag(),
			Params: params,
		})
	}

	return NewValidationError(fields...)
}
