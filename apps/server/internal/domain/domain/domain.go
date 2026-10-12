package domain

import (
	"time"
	"uuid"
)

type Domain struct {
	ID        uuid.UUID
	Domain    string
	DeletedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
