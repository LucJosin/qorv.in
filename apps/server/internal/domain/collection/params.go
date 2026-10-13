package collection

import (
	"uuid"

	"github.com/lucjosin/qorv.in/internal/pagination"
)

// Params contains collection filters and pagination. Only filters are stored in cursors.
type Params struct {
	pagination.Query `json:"-"`
	WorkspaceID      uuid.UUID `json:"workspace_id,omitzero"`
}
