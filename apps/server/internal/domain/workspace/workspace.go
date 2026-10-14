package workspace

import (
	"strings"
	"time"
	"uuid"

	"github.com/lucjosin/qorv.in/internal/validatorx"
)

type Workspace struct {
	ID          uuid.UUID
	DomainID    uuid.UUID
	Slug        string
	Name        string
	Description string
	Color       string
	DeletedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (w *Workspace) ValidateAndPrepare() error {
	w.Prepare()
	return w.Validate()
}

func (w *Workspace) Validate() error {
	var fields []validatorx.FieldError

	slug := strings.TrimSpace(w.Slug)
	if slug == "" {
		field := validatorx.Field("slug", "is required")
		field.Tag = "required"
		fields = append(fields, field)
	}

	return validatorx.NewValidationError(fields...)
}

func (w *Workspace) Prepare() {
}
