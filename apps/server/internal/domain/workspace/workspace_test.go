package workspace_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lucjosin/qorv.in/internal/domain/workspace"
	"github.com/lucjosin/qorv.in/internal/errs"
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

func TestWorkspaceValidate_Slug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		invalid bool
	}{
		{
			name: "valid",
			slug: "main",
		},
		{
			name: "surrounding whitespace",
			slug: " main ",
		},
		{
			name:    "empty",
			invalid: true,
		},
		{
			name:    "whitespace",
			slug:    " \t\n",
			invalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := workspace.Workspace{
				Slug: tt.slug,
			}

			err := entity.Validate()

			var wantFields []validatorx.FieldError
			if tt.invalid {
				wantFields = []validatorx.FieldError{
					{
						Field:   "slug",
						Tag:     "required",
						Message: "is required",
					},
				}
			}
			assertValidationError(t, err, wantFields)

			want := tt.slug
			got := entity.Slug
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("slug changed (-want +got):\n%s", diff)
			}
		})
	}
}

func assertValidationError(t *testing.T, err error, want []validatorx.FieldError) {
	t.Helper()

	if want == nil {
		if err != nil {
			t.Fatal(err)
		}
		return
	}

	isInvalid := errors.Is(err, errs.ErrInvalid)
	if !isInvalid {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}

	validation, ok := errors.AsType[*validatorx.ValidationError](err)
	if !ok {
		t.Fatalf("error = %v, want field validation errors", err)
	}

	got := validation.Fields
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("fields mismatch (-want +got):\n%s", diff)
	}
}
