package collection

import (
	"context"

	"github.com/lucjosin/qorv.in/internal/pagination"
)

type Service interface {
	List(ctx context.Context, params Params) (pagination.Page[Collection], error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) List(ctx context.Context, params Params) (pagination.Page[Collection], error) {
	err := params.Validate()
	if err != nil {
		return pagination.Page[Collection]{}, err
	}
	items, err := s.repo.List(ctx, params)
	if err != nil {
		return pagination.Page[Collection]{}, err
	}
	return pagination.NewPage(items, params.Limit), nil
}
