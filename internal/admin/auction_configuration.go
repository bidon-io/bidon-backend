package admin

import (
	"context"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/auction"
)

type AuctionConfiguration struct {
	ID int64 `json:"id"`
	AuctionConfigurationAttrs
	App     App      `json:"app"`
	Segment *Segment `json:"segment"`
}

// AuctionConfigurationAttrs is attributes of Configuration. Used to create and update configurations
type AuctionConfigurationAttrs struct {
	Name                     string                `json:"name"`
	AppID                    int64                 `json:"app_id"`
	AdType                   ad.Type               `json:"ad_type"`
	Rounds                   []auction.RoundConfig `json:"rounds"`
	Pricefloor               float64               `json:"pricefloor"`
	SegmentID                *int64                `json:"segment_id"`
	ExternalWinNotifications *bool                 `json:"external_win_notifications"`
}

type AuctionConfigurationService = ResourceService[AuctionConfiguration, AuctionConfigurationAttrs]

func NewAuctionConfigurationService(store Store) *AuctionConfigurationService {
	return &AuctionConfigurationService{
		repo: store.AuctionConfigurations(),
		policy: &auctionConfigurationPolicy{
			repo: store.AuctionConfigurations(),
		},
	}
}

type AuctionConfigurationRepo interface {
	ResourceRepo[AuctionConfiguration, AuctionConfigurationAttrs]

	ListOwnedByUser(ctx context.Context, userID int64) ([]AuctionConfiguration, error)
	FindOwnedByUser(ctx context.Context, userID, id int64) (*AuctionConfiguration, error)
}

type auctionConfigurationPolicy struct {
	repo AuctionConfigurationRepo
}

func (p *auctionConfigurationPolicy) scope(authCtx AuthContext) (resourceScope[AuctionConfiguration], error) {
	return &auctionConfigurationScope{
		repo:    p.repo,
		authCtx: authCtx,
	}, nil
}

type auctionConfigurationScope struct {
	repo AuctionConfigurationRepo

	authCtx AuthContext
}

func (s *auctionConfigurationScope) list(ctx context.Context) ([]AuctionConfiguration, error) {
	if s.authCtx.IsAdmin() {
		return s.repo.List(ctx)
	}

	return s.repo.ListOwnedByUser(ctx, s.authCtx.UserID())
}

func (s *auctionConfigurationScope) find(ctx context.Context, id int64) (*AuctionConfiguration, error) {
	if s.authCtx.IsAdmin() {
		return s.repo.Find(ctx, id)
	}

	return s.repo.FindOwnedByUser(ctx, s.authCtx.UserID(), id)
}
