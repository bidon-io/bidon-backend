package dspsim

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/db"
)

// CatalogLineItem is a bidding line item (ad unit) attached to an auction.
type CatalogLineItem struct {
	ID     int64     `json:"id"`
	UID    string    `json:"uid"`
	Label  string    `json:"label"`
	Demand string    `json:"demand"`
	Format ad.Format `json:"format"`
	Width  int32     `json:"width"`
	Height int32     `json:"height"`
}

// ConfiguredAuction is one auction_configurations row, resolved against its app
// and its bidding line items. It is what the matcher looks a bid request up in.
type ConfiguredAuction struct {
	ConfigID   int64             `json:"config_id"`
	AppID      int64             `json:"app_id"`
	Bundle     string            `json:"bundle"`
	Platform   string            `json:"platform"`
	AdType     ad.Type           `json:"ad_type"`
	PriceFloor float64           `json:"pricefloor"`
	Demands    []string          `json:"bidding_demands"`
	Formats    []ad.Format       `json:"formats"`
	LineItems  []CatalogLineItem `json:"line_items"`
}

// HasDemand reports whether key is configured as bidding demand for the
// auction, either on the configuration itself or through a line item.
func (a *ConfiguredAuction) HasDemand(key string) bool {
	key = strings.ToLower(key)
	for _, d := range a.Demands {
		if strings.ToLower(d) == key {
			return true
		}
	}
	for _, li := range a.LineItems {
		if strings.ToLower(li.Demand) == key {
			return true
		}
	}
	return false
}

// SupportsFormat reports whether the auction has a line item able to serve the
// requested format. It mirrors the widening the SDK API applies when matching
// ad units (see internal/auction/store/ad_units_matcher.go selectAdFormats):
// ADAPTIVE is interchangeable with BANNER and LEADERBOARD, MREC is not.
func (a *ConfiguredAuction) SupportsFormat(f ad.Format) bool {
	if a.AdType != ad.BannerType || f == ad.EmptyFormat {
		return true
	}
	for _, have := range a.Formats {
		if have == f || formatsInterchangeable(have, f) {
			return true
		}
	}
	return false
}

func formatsInterchangeable(a, b ad.Format) bool {
	interchangeable := func(x, y ad.Format) bool {
		return x == ad.AdaptiveFormat && (y == ad.BannerFormat || y == ad.LeaderboardFormat)
	}
	return interchangeable(a, b) || interchangeable(b, a)
}

// Catalog is an immutable snapshot of the configured auctions, indexed by app
// bundle. Snapshots are swapped wholesale by CatalogStore.
type Catalog struct {
	LoadedAt time.Time
	byBundle map[string][]*ConfiguredAuction
	auctions []*ConfiguredAuction
}

// Lookup returns the auctions configured for a bundle and ad type.
func (c *Catalog) Lookup(bundle string, adType ad.Type) []*ConfiguredAuction {
	if c == nil {
		return nil
	}
	var out []*ConfiguredAuction
	for _, a := range c.byBundle[strings.ToLower(bundle)] {
		if a.AdType == adType {
			out = append(out, a)
		}
	}
	return out
}

// KnowsBundle reports whether any auction is configured for the bundle.
func (c *Catalog) KnowsBundle(bundle string) bool {
	if c == nil {
		return false
	}
	return len(c.byBundle[strings.ToLower(bundle)]) > 0
}

// Auctions returns every auction in the snapshot, for /debug/catalog.
func (c *Catalog) Auctions() []*ConfiguredAuction {
	if c == nil {
		return nil
	}
	return c.auctions
}

// Bundles returns the distinct bundles in the snapshot.
func (c *Catalog) Bundles() []string {
	if c == nil {
		return nil
	}
	out := make([]string, 0, len(c.byBundle))
	for b := range c.byBundle {
		out = append(out, b)
	}
	sort.Strings(out)
	return out
}

// auctionRow is the projection of auction_configurations joined with apps.
type auctionRow struct {
	ID          int64
	AppID       int64
	AdType      db.AdType
	Pricefloor  float64
	Bidding     pq.StringArray `gorm:"type:character varying[]"`
	AdUnitIds   pq.Int64Array  `gorm:"type:bigint[]"`
	PackageName string
	PlatformID  db.PlatformID
}

// lineItemRow is the projection of bidding line items joined with their demand
// source.
type lineItemRow struct {
	ID        int64
	AppID     int64
	AdType    db.AdType
	Format    string
	Width     int32
	Height    int32
	PublicUID int64
	HumanName string
	APIKey    string
}

// CatalogStore loads catalog snapshots from Postgres and hands the current one
// out to readers. Reads are lock-free; a refresh swaps an atomic pointer.
type CatalogStore struct {
	DB     *db.DB
	Logger *zap.Logger

	current atomic.Pointer[Catalog]
}

// NewCatalogStore returns a store with an empty snapshot already installed, so
// Get never returns nil before the first refresh completes.
func NewCatalogStore(database *db.DB, logger *zap.Logger) *CatalogStore {
	s := &CatalogStore{DB: database, Logger: logger}
	s.current.Store(&Catalog{byBundle: map[string][]*ConfiguredAuction{}})
	return s
}

// Get returns the current snapshot.
func (s *CatalogStore) Get() *Catalog {
	return s.current.Load()
}

// Refresh reloads the catalog from Postgres. On failure the previous snapshot
// is kept, so a database blip does not stop the simulator from bidding.
func (s *CatalogStore) Refresh(ctx context.Context) error {
	var auctions []auctionRow
	err := s.DB.WithContext(ctx).
		Table("auction_configurations as ac").
		Select("ac.id, ac.app_id, ac.ad_type, ac.pricefloor, ac.bidding, ac.ad_unit_ids, apps.package_name, apps.platform_id").
		Joins("JOIN apps ON apps.id = ac.app_id").
		Where("ac.deleted_at IS NULL").
		Where("apps.deleted_at IS NULL").
		Where("apps.package_name IS NOT NULL AND apps.package_name <> ''").
		Scan(&auctions).Error
	if err != nil {
		return err
	}

	var lineItems []lineItemRow
	err = s.DB.WithContext(ctx).
		Table("line_items as li").
		Select(`li.id, li.app_id, li.ad_type, COALESCE(li.format, '') as format, li.width, li.height,
			COALESCE(li.public_uid, 0) as public_uid, li.human_name, ds.api_key`).
		Joins("JOIN demand_source_accounts dsa ON dsa.id = li.account_id").
		Joins("JOIN demand_sources ds ON ds.id = dsa.demand_source_id").
		Where("li.deleted_at IS NULL").
		Where("li.bidding IS TRUE").
		Scan(&lineItems).Error
	if err != nil {
		return err
	}

	catalog := buildCatalog(auctions, lineItems)
	s.current.Store(catalog)

	if s.Logger != nil {
		s.Logger.Info("dspsim: catalog refreshed",
			zap.Int("auctions", len(catalog.auctions)),
			zap.Int("bundles", len(catalog.byBundle)),
			zap.Int("line_items", len(lineItems)),
		)
	}
	return nil
}

// Run refreshes the catalog on a ticker until ctx is cancelled.
func (s *CatalogStore) Run(ctx context.Context, every time.Duration) {
	if every <= 0 {
		return
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Refresh(ctx); err != nil && s.Logger != nil {
				s.Logger.Error("dspsim: catalog refresh failed, keeping previous snapshot", zap.Error(err))
			}
		}
	}
}

// buildCatalog joins the two projections in memory. It is pure so it can be
// tested without a database.
func buildCatalog(auctions []auctionRow, lineItems []lineItemRow) *Catalog {
	// Index line items by app + ad type, and by id for ad_unit_ids filtering.
	type appAdType struct {
		appID  int64
		adType db.AdType
	}
	byApp := map[appAdType][]lineItemRow{}
	for _, li := range lineItems {
		k := appAdType{li.AppID, li.AdType}
		byApp[k] = append(byApp[k], li)
	}

	catalog := &Catalog{
		LoadedAt: time.Now().UTC(),
		byBundle: map[string][]*ConfiguredAuction{},
	}

	for _, row := range auctions {
		adType := row.AdType.Domain()
		if adType == ad.UnknownType {
			continue
		}

		allowed := map[int64]bool{}
		for _, id := range row.AdUnitIds {
			allowed[id] = true
		}

		auction := &ConfiguredAuction{
			ConfigID:   row.ID,
			AppID:      row.AppID,
			Bundle:     row.PackageName,
			Platform:   platformName(row.PlatformID),
			AdType:     adType,
			PriceFloor: row.Pricefloor,
			Demands:    []string(row.Bidding),
		}

		formats := map[ad.Format]bool{}
		for _, li := range byApp[appAdType{row.AppID, row.AdType}] {
			if len(allowed) > 0 && !allowed[li.ID] {
				continue
			}
			format := ad.Format(strings.ToUpper(li.Format))
			formats[format] = true
			auction.LineItems = append(auction.LineItems, CatalogLineItem{
				ID:     li.ID,
				UID:    strconv.FormatInt(li.PublicUID, 10),
				Label:  li.HumanName,
				Demand: li.APIKey,
				Format: format,
				Width:  li.Width,
				Height: li.Height,
			})
		}

		for f := range formats {
			if f != ad.EmptyFormat {
				auction.Formats = append(auction.Formats, f)
			}
		}
		sort.Slice(auction.Formats, func(i, j int) bool { return auction.Formats[i] < auction.Formats[j] })

		key := strings.ToLower(auction.Bundle)
		catalog.byBundle[key] = append(catalog.byBundle[key], auction)
		catalog.auctions = append(catalog.auctions, auction)
	}

	return catalog
}

func platformName(id db.PlatformID) string {
	switch id {
	case db.AndroidPlatformID:
		return "android"
	case db.IOSPlatformID:
		return "ios"
	default:
		return "unknown"
	}
}
