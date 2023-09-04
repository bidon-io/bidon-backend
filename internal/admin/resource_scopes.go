package admin

import (
	"context"
	"errors"
)

type publicResourceScope[Resource any] struct {
	repo interface {
		List(context.Context) ([]Resource, error)
		Find(context.Context, int64) (*Resource, error)
	}
}

func (s *publicResourceScope[Resource]) list(ctx context.Context) ([]Resource, error) {
	return s.repo.List(ctx)
}

func (s *publicResourceScope[Resource]) find(ctx context.Context, id int64) (*Resource, error) {
	return s.repo.Find(ctx, id)
}

type privateResourceScope[Resource any] struct {
	repo interface {
		List(context.Context) ([]Resource, error)
		Find(context.Context, int64) (*Resource, error)
	}

	authCtx AuthContext
}

func (s *privateResourceScope[Resource]) list(ctx context.Context) ([]Resource, error) {
	if !s.authCtx.IsAdmin() {
		return nil, errors.New("unauthorized")
	}

	return s.repo.List(ctx)
}

func (s *privateResourceScope[Resource]) find(ctx context.Context, id int64) (*Resource, error) {
	if !s.authCtx.IsAdmin() {
		return nil, errors.New("unauthorized")
	}

	return s.repo.Find(ctx, id)
}

type ownedResourceScope[Resource any] struct {
	repo interface {
		List(context.Context) ([]Resource, error)
		ListOwnedByUser(context.Context, int64) ([]Resource, error)
		Find(context.Context, int64) (*Resource, error)
		FindOwnedByUser(context.Context, int64, int64) (*Resource, error)
	}

	authCtx AuthContext
}

func (s *ownedResourceScope[Resource]) list(ctx context.Context) ([]Resource, error) {
	if s.authCtx.IsAdmin() {
		return s.repo.List(ctx)
	}

	return s.repo.ListOwnedByUser(ctx, s.authCtx.UserID())
}

func (s *ownedResourceScope[Resource]) find(ctx context.Context, id int64) (*Resource, error) {
	if s.authCtx.IsAdmin() {
		return s.repo.Find(ctx, id)
	}

	return s.repo.FindOwnedByUser(ctx, s.authCtx.UserID(), id)
}

type ownedOrSharedResourceScope[Resource any] struct {
	repo interface {
		List(context.Context) ([]Resource, error)
		ListOwnedByUserOrShared(context.Context, int64) ([]Resource, error)
		Find(context.Context, int64) (*Resource, error)
		FindOwnedByUserOrShared(context.Context, int64, int64) (*Resource, error)
	}

	authCtx AuthContext
}

func (s *ownedOrSharedResourceScope[Resource]) list(ctx context.Context) ([]Resource, error) {
	if s.authCtx.IsAdmin() {
		return s.repo.List(ctx)
	}

	return s.repo.ListOwnedByUserOrShared(ctx, s.authCtx.UserID())
}

func (s *ownedOrSharedResourceScope[Resource]) find(ctx context.Context, id int64) (*Resource, error) {
	if s.authCtx.IsAdmin() {
		return s.repo.Find(ctx, id)
	}

	return s.repo.FindOwnedByUserOrShared(ctx, s.authCtx.UserID(), id)
}
