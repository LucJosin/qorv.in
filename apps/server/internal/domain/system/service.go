package system

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/lucjosin/qorv.in/internal/errs"
)

type Service interface {
	Info(ctx context.Context) (System, error)
	Configure(ctx context.Context, ownerUserID uuid.UUID) (System, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) Info(ctx context.Context) (System, error) {
	return s.repo.FindOne(ctx)
}

func (s *service) Configure(ctx context.Context, ownerUserID uuid.UUID) (System, error) {
	entity, err := s.repo.Create(ctx, ownerUserID)
	if err != nil {
		if !errors.Is(err, errs.ErrConflict) {
			return System{}, err
		}
		entity, err = s.repo.FindOne(ctx)
		if err != nil {
			return System{}, fmt.Errorf("retrieving system after conflict: %w", err)
		}
	}
	if entity.OwnerUserID != ownerUserID {
		return System{}, fmt.Errorf("%w: system already has a different owner", errs.ErrConflict)
	}
	return entity, nil
}
