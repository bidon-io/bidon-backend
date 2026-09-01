package dspsim

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
)

// Macros the simulator embeds in its notification URLs. bidon substitutes a
// query param only when the whole value is one of these tokens, so each macro
// is the entire value of its own param (see
// internal/notification/event_sender.go).
const (
	MacroPrice     = "${AUCTION_PRICE}"
	MacroMinToWin  = "${AUCTION_MIN_TO_WIN}"
	MacroAuctionID = "${AUCTION_ID}"
	MacroBidID     = "${AUCTION_BID_ID}"
	MacroImpID     = "${AUCTION_IMP_ID}"
	MacroSeatID    = "${AUCTION_SEAT_ID}"
	MacroAdID      = "${AUCTION_AD_ID}"
	MacroLoss      = "${AUCTION_LOSS}"
	MacroCurrency  = "${AUCTION_CURRENCY}"
)

// Bidder turns a match into an OpenRTB bid response plus the record the
// simulator keeps about it.
type Bidder struct {
	Config  Config
	Library *Library

	mu  sync.Mutex
	rnd *rand.Rand
}

// NewBidder returns a bidder seeded from cfg. A zero seed uses the clock.
func NewBidder(cfg Config, lib *Library) *Bidder {
	seed := cfg.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &Bidder{
		Config:  cfg,
		Library: lib,
		rnd:     rand.New(rand.NewSource(seed)), //nolint:gosec // deterministic test fixtures, not crypto
	}
}

// SetLibrary swaps the creative library, so /debug/reload can pick up edits
// without a restart.
func (b *Bidder) SetLibrary(lib *Library) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Library = lib
}

// GetLibrary returns the current creative library.
func (b *Bidder) GetLibrary() *Library {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Library
}

// ShouldSkip reports whether this request falls into the configured random
// no-bid rate.
func (b *Bidder) ShouldSkip() bool {
	if b.Config.NoBidRate <= 0 {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rnd.Float64() < b.Config.NoBidRate
}

// Build produces the bid response for a match. forcedCreative pins a creative
// by id. It returns a no-bid reason when no creative can serve the slot.
func (b *Bidder) Build(match *Match, forcedCreative string) (*openrtb2.BidResponse, *BidRecord, NoBidReason, error) {
	summary := match.Summary
	library := b.GetLibrary()

	selection, ok := b.selectCreative(library, summary, forcedCreative)
	if !ok {
		return nil, nil, ReasonNoCreative, nil
	}
	creative := selection.Creative

	bidID, err := uuid.NewV4()
	if err != nil {
		return nil, nil, "", fmt.Errorf("generate bid id: %w", err)
	}
	id := bidID.String()

	price := b.price(summary.Floor)
	w, h := creativeSize(creative, summary)

	base := b.Config.PublicURL
	data := CreativeData{
		BidID:         id,
		ImpID:         summary.ImpID,
		RequestID:     summary.RequestID,
		DSP:           summary.DSP,
		CreativeID:    creative.ID,
		Currency:      b.Config.Currency,
		Price:         price,
		W:             w,
		H:             h,
		PublicURL:     base,
		AssetURL:      base + "/creative/asset",
		ClickURL:      fmt.Sprintf("%s/creative/click/%s", base, id),
		ImpressionURL: fmt.Sprintf("%s/creative/impression/%s", base, id),
		TrackURL:      fmt.Sprintf("%s/creative/track/%s", base, id),
	}

	adm, err := creative.Render(data)
	if err != nil {
		return nil, nil, "", err
	}

	record := &BidRecord{
		BidID:            id,
		RequestID:        summary.RequestID,
		ImpID:            summary.ImpID,
		Bundle:           summary.Bundle,
		DSP:              summary.DSP,
		AdType:           summary.AdType,
		Format:           summary.Format,
		Width:            w,
		Height:           h,
		Floor:            summary.Floor,
		Price:            price,
		Currency:         b.Config.Currency,
		CreativeID:       creative.ID,
		CreativeType:     creative.Type,
		CreativeBucket:   selection.Bucket,
		AuctionConfigID:  match.Auction.ConfigID,
		DemandConfigured: match.DemandConfigured,
		CreatedAt:        time.Now().UTC(),
		NURL:             b.winURL(id),
		BURL:             b.billingURL(id),
		LURL:             b.lossURL(id),
	}

	response := &openrtb2.BidResponse{
		ID:    summary.RequestID,
		BidID: id,
		Cur:   b.Config.Currency,
		SeatBid: []openrtb2.SeatBid{{
			Seat: b.Config.Seat,
			Bid: []openrtb2.Bid{{
				ID:      id,
				ImpID:   summary.ImpID,
				Price:   price,
				AdM:     adm,
				AdID:    creative.ID,
				CrID:    creative.ID,
				CID:     creative.CampID,
				ADomain: creative.ADomain,
				Bundle:  creative.Bundle,
				W:       w,
				H:       h,
				NURL:    record.NURL,
				BURL:    record.BURL,
				LURL:    record.LURL,
			}},
		}},
	}

	library.MarkServed(selection.Bucket, creative.ID)

	return response, record, "", nil
}

func (b *Bidder) selectCreative(library *Library, summary RequestSummary, forcedCreative string) (*Selection, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return library.Select(summary.DSP, summary.AdType, summary.Format, summary.Width, summary.Height, forcedCreative, b.rnd)
}

// price draws a value above the impression floor, capped at MaxPrice.
func (b *Bidder) price(floor float64) float64 {
	if floor <= 0 {
		floor = b.Config.FallbackFloor
	}

	b.mu.Lock()
	factor := b.Config.PriceMultMin + b.rnd.Float64()*(b.Config.PriceMultMax-b.Config.PriceMultMin)
	b.mu.Unlock()

	price := floor * factor
	if price > b.Config.MaxPrice {
		price = b.Config.MaxPrice
	}
	price = math.Round(price*10_000) / 10_000

	// Guard the rounding and the cap: a bid at or below the floor is useless.
	if price <= floor {
		price = math.Round((floor+0.01)*10_000) / 10_000
	}
	return price
}

// creativeSize prefers the creative's declared size, falling back to the size
// asked for in the impression.
func creativeSize(c *Creative, summary RequestSummary) (int64, int64) {
	if c.Width > 0 && c.Height > 0 {
		return c.Width, c.Height
	}
	return summary.Width, summary.Height
}

func (b *Bidder) winURL(bidID string) string {
	return b.notifyURL("win", bidID, []string{
		"price=" + MacroPrice,
		"mintowin=" + MacroMinToWin,
		"auction=" + MacroAuctionID,
		"bid=" + MacroBidID,
		"imp=" + MacroImpID,
		"seat=" + MacroSeatID,
		"ad=" + MacroAdID,
		"cur=" + MacroCurrency,
	})
}

func (b *Bidder) billingURL(bidID string) string {
	return b.notifyURL("billing", bidID, []string{
		"price=" + MacroPrice,
		"bid=" + MacroBidID,
		"imp=" + MacroImpID,
		"cur=" + MacroCurrency,
	})
}

func (b *Bidder) lossURL(bidID string) string {
	return b.notifyURL("loss", bidID, []string{
		"price=" + MacroPrice,
		"loss=" + MacroLoss,
		"mintowin=" + MacroMinToWin,
		"auction=" + MacroAuctionID,
		"bid=" + MacroBidID,
		"cur=" + MacroCurrency,
	})
}

// notifyURL writes the query unencoded, the way real DSPs emit macros, so the
// URL stays readable in logs. bidon parses it with url.ParseQuery, which
// accepts the braces.
func (b *Bidder) notifyURL(kind, bidID string, params []string) string {
	return fmt.Sprintf("%s/notify/%s/%s?%s", b.Config.PublicURL, kind, bidID, strings.Join(params, "&"))
}

// AdTypeString renders an ad type for logs and debug output.
func AdTypeString(t ad.Type) string {
	if t == ad.UnknownType {
		return "unknown"
	}
	return string(t)
}
