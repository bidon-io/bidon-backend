package notification

import (
	"context"

	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/notification/store"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type Notification struct {
	RoundID string
}

type Handler struct {
	AuctionResultRepo AuctionResultRepo
}

type AuctionResultRepo interface {
	CreateOrUpdate(ctx context.Context, imp *schema.Imp, bids []store.Bid) error
}

// HandleRound is used to handle bidding round, it is called after all adapters have responded with bids or errors
// Results saved to redis
func (h Handler) HandleRound(ctx context.Context, imp *schema.Imp, responses []*adapters.DemandResponse) error {
	var bids []store.Bid
	for _, resp := range responses {
		if resp.IsBid() {
			bids = append(bids, store.Bid{
				ID:       resp.Bid.ID,
				ImpID:    resp.Bid.ImpID,
				Price:    resp.Bid.Price,
				Payload:  resp.Bid.Payload,
				DemandID: resp.Bid.DemandID,
				AdID:     resp.Bid.AdID,
				SeatID:   resp.Bid.SeatID,
				LURL:     resp.Bid.LURL,
				NURL:     resp.Bid.NURL,
				BURL:     resp.Bid.BURL,
			})
		}
	}

	return h.AuctionResultRepo.CreateOrUpdate(ctx, imp, bids)
}

// HandleStats is used to handle /stats request
// Finalize results of auction in redis
// If external_win_notification is enabled - do nothing, wait /win or /loss request
// If external_win_notification is disabled - send win/loss notifications to demands
func (h Handler) HandleStats(ctx context.Context, imp *schema.Imp, responses []*adapters.DemandResponse) error {
	return nil
}

// HandleShow is used to handle impressions
// Send burl to demand
func (h *Handler) HandleShow(adaprts []*adapters.DemandResponse) error {
	return nil
}

// If external_win_notification is enabled - send win/loss notifications to demands
// If external_win_notification is disabled - do nothing
func (h *Handler) HandleWin(adaprts []*adapters.DemandResponse) error {
	return nil
}

// If external_win_notification is enabled - send win/loss notifications to demands
// If external_win_notification is disabled - do nothing
func (h *Handler) HandleLoss(adaprts []*adapters.DemandResponse) error {
	return nil
}
