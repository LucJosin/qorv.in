package user

import (
	"time"
	"uuid"
)

type User struct {
	ID            uuid.UUID
	FirstName     string
	LastName      string
	Username      string
	Email         string
	Password      string
	EmailVerified bool
	AvatarURL     *string
	DeletedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (u User) IsZero() bool {
	return u.ID == uuid.Nil()
}
