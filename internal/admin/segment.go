// Package admin implements an HTTP API handlers for managing entities.
package admin

import (
	"context"

	"github.com/bidon-io/bidon-backend/internal/segment"
)

type Segment struct {
	ID int64 `json:"id"`
	SegmentAttrs
	App `json:"app"`
}

type SegmentAttrs struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Filters     []segment.Filter `json:"filters"`
	Enabled     *bool            `json:"enabled"`
	AppID       int64            `json:"app_id"`
	Priority    int32            `json:"priority"`
}

type SegmentService = ResourceService[Segment, SegmentAttrs]

func NewSegmentService(store Store) *SegmentService {
	return &SegmentService{
		repo: store.Segments(),
		policy: &segmentPolicy{
			repo: store.Segments(),
		},
	}
}

type SegmentRepo interface {
	ResourceRepo[Segment, SegmentAttrs]

	ListOwnedByUser(ctx context.Context, userID int64) ([]Segment, error)
	FindOwnedByUser(ctx context.Context, userID, id int64) (*Segment, error)
}

type segmentPolicy struct {
	repo SegmentRepo
}

func (p *segmentPolicy) scope(authCtx AuthContext) (resourceScope[Segment], error) {
	return &segmentScope{
		repo:    p.repo,
		authCtx: authCtx,
	}, nil
}

type segmentScope struct {
	repo SegmentRepo

	authCtx AuthContext
}

func (s *segmentScope) list(ctx context.Context) ([]Segment, error) {
	if s.authCtx.IsAdmin() {
		return s.repo.List(ctx)
	}

	return s.repo.ListOwnedByUser(ctx, s.authCtx.UserID())
}

func (s *segmentScope) find(ctx context.Context, id int64) (*Segment, error) {
	if s.authCtx.IsAdmin() {
		return s.repo.Find(ctx, id)
	}

	return s.repo.FindOwnedByUser(ctx, s.authCtx.UserID(), id)
}
