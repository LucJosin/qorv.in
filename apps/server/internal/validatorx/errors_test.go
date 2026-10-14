package validatorx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/errs"
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

func TestNewValidationError_Wrapped(t *testing.T) {
	field := validatorx.Field("limit", "out of range")
	validationError := validatorx.NewValidationError(field)
	err := fmt.Errorf("validating request: %w", validationError)

	isInvalid := errors.Is(err, errs.ErrInvalid)
	if !isInvalid {
		t.Fatal("validation error does not wrap ErrInvalid")
	}

	validation, ok := errors.AsType[*validatorx.ValidationError](err)
	if !ok {
		t.Fatal("wrapped field errors cannot be extracted")
	}

	want := []validatorx.FieldError{
		{
			Field:   "limit",
			Tag:     "invalid",
			Message: "out of range",
		},
	}

	got := validation.Fields
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("fields mismatch (-want +got):\n%s", diff)
	}

	wantMessage := errs.ErrInvalid.Error()
	gotMessage := validation.Error()
	if diff := cmp.Diff(wantMessage, gotMessage); diff != "" {
		t.Fatalf("message mismatch (-want +got):\n%s", diff)
	}
}

func TestNewValidationError_Empty(t *testing.T) {
	err := validatorx.NewValidationError()
	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
}
