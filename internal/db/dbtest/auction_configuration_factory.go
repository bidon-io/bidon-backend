package dbtest

import (
	"database/sql"
	"fmt"
	"testing"

	"gorm.io/datatypes"

	"github.com/bidon-io/bidon-backend/internal/db"
)

func auctionConfigurationDefaults(n uint32) func(*db.AuctionConfiguration) {
	return func(config *db.AuctionConfiguration) {
		if config.AppID == 0 && config.App.ID == 0 {
			config.App = BuildApp(func(app *db.App) {
				*app = config.App
			})
		}
		if config.AdType == 0 {
			config.AdType = db.BannerAdType
		}
		if config.Name == (sql.NullString{}) {
			config.Name = sql.NullString{
				String: fmt.Sprintf("Test Auction Configuration %d", n),
				Valid:  true,
			}
		}
		if config.Rounds == nil {
			config.Rounds = datatypes.JSON("[]")
		}
		if config.Status == (sql.NullInt32{}) {
			config.Status = sql.NullInt32{Int32: 1, Valid: true}
		}
		if config.Settings == nil {
			// Match reads this to select the v2 auction config for an app+ad_type
			// (see internal/auction/store/config_fetcher.go).
			config.Settings = map[string]any{"v2": true}
		}
		if config.Timeout == 0 {
			config.Timeout = 30000
		}
		if config.PublicUID == (sql.NullInt64{}) {
			config.PublicUID = sql.NullInt64{
				Int64: int64(n),
				Valid: true,
			}
		}
		if config.IsDefault == nil {
			isDefault := false
			config.IsDefault = &isDefault
		}
		if config.ExternalWinNotifications == nil {
			externalWinNotifications := false
			config.ExternalWinNotifications = &externalWinNotifications
		}
	}
}

func BuildAuctionConfiguration(opts ...func(*db.AuctionConfiguration)) db.AuctionConfiguration {
	var config db.AuctionConfiguration

	n := counter.get("auction_configuration")

	opts = append(opts, auctionConfigurationDefaults(n))
	for _, opt := range opts {
		opt(&config)
	}

	return config
}

func CreateAuctionConfiguration(t *testing.T, tx *db.DB, opts ...func(*db.AuctionConfiguration)) db.AuctionConfiguration {
	t.Helper()

	config := BuildAuctionConfiguration(opts...)
	if err := tx.Create(&config).Error; err != nil {
		t.Fatalf("Failed to create auction configuration: %v", err)
	}

	return config
}
