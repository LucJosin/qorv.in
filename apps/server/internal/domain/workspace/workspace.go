package workspace

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/lucjosin/qorv.in/internal/errs"
)

type Workspace struct {
	ID          uuid.UUID
	PublicID    uuid.UUID
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
	if strings.TrimSpace(w.Slug) == "" {
		return fmt.Errorf("%w: workspace slug is required", errs.ErrInvalid)
	}
	return nil
}

func (w *Workspace) Prepare() {
}
