package validatorx_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/errs"
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

func TestValidate_FieldErrors(t *testing.T) {
	v := validatorx.New()

	tests := []struct {
		name  string
		value Request
		want  []validatorx.FieldError
	}{
		{
			name: "valid",
			value: Request{
				Profile: Profile{
					Name: "Owner",
				},
				Count: 1,
			},
		},
		{
			name: "multiple and nested fields",
			value: Request{
				Count: 0,
			},
			want: []validatorx.FieldError{
				{
					Field: "name",
					Tag:   "required",
				},
				{
					Field:  "count",
					Tag:    "min",
					Params: []string{"1"},
				},
			},
		},
		{
			name: "max and invalid value",
			value: Request{
				Profile: Profile{
					Name: "Owner",
				},
				Count: 11,
				Email: "invalid",
			},
			want: []validatorx.FieldError{
				{
					Field:  "count",
					Tag:    "max",
					Params: []string{"10"},
				},
				{
					Field: "email",
					Tag:   "email",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(&tt.value)

			if tt.want == nil {
				if err != nil {
					t.Fatal(err)
				}

				return
			}

			isInvalid := errors.Is(err, errs.ErrInvalid)
			if !isInvalid {
				t.Fatalf("error = %v, want invalid input", err)
			}

			validation, ok := errors.AsType[*validatorx.ValidationError](err)
			if !ok {
				t.Fatalf("error = %v, want field failures", err)
			}

			want := tt.want
			got := validation.Fields
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("fields mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidate_InvalidInput(t *testing.T) {
	v := validatorx.New()

	tests := []struct {
		name  string
		value any
	}{
		{
			name: "nil",
		},
		{
			name:  "scalar",
			value: 1,
		},
		{
			name:  "nil pointer",
			value: (*Request)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.value)

			isInvalid := errors.Is(err, errs.ErrInvalid)
			if err == nil || isInvalid {
				t.Fatalf("error = %v, want an internal validation error", err)
			}

			_, ok := errors.AsType[*validatorx.ValidationError](err)
			if ok {
				t.Fatal("internal error converted to field validation failure")
			}
		})
	}
}

func TestValidate_AnonymousStructFields(t *testing.T) {
	v := validatorx.New()

	value := struct {
		Profiles []Profile `json:"profiles" validate:"dive"`
	}{
		Profiles: []Profile{
			{
				Name: "",
			},
		},
	}

	err := v.Validate(value)

	validation, ok := errors.AsType[*validatorx.ValidationError](err)
	if !ok {
		t.Fatalf("error = %v, want field failures", err)
	}

	want := []validatorx.FieldError{
		{
			Field: "name",
			Tag:   "required",
		},
	}

	got := validation.Fields
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("fields mismatch (-want +got):\n%s", diff)
	}
}

type Profile struct {
	Name string `json:"name,omitempty" validate:"required"`
}

type Request struct {
	Profile Profile `json:"profile"`
	Count   int     `json:"count" validate:"min=1,max=10"`
	Email   string  `json:"email" validate:"omitempty,email"`
	Ignored string  `json:"-" validate:"-"`
}
