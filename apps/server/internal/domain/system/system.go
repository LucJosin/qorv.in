package system

import (
	"time"
	"uuid"
)

type System struct {
	OwnerUserID uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
