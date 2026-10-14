package workspace_test

import (
	"testing"

	"github.com/lucjosin/qorv.in/internal/domain/workspace"
	"github.com/lucjosin/qorv.in/internal/validatorx"
)

func TestWorkspaceUserValidate_Role(t *testing.T) {
	tests := []struct {
		name    string
		role    workspace.WorkspaceRole
		invalid bool
	}{
		{
			name: "owner",
			role: workspace.OwnerRole,
		},
		{
			name: "admin",
			role: workspace.AdminRole,
		},
		{
			name: "member",
			role: workspace.MemberRole,
		},
		{
			name: "viewer",
			role: workspace.ViewerRole,
		},
		{
			name:    "empty",
			invalid: true,
		},
		{
			name:    "unknown",
			role:    "unknown",
			invalid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := workspace.WorkspaceUser{
				Role: tt.role,
			}

			err := entity.Validate()

			var want []validatorx.FieldError
			if tt.invalid {
				want = []validatorx.FieldError{
					{
						Field:   "role",
						Tag:     "invalid",
						Message: "must be a valid workspace role",
					},
				}
			}
			assertValidationError(t, err, want)
		})
	}
}
