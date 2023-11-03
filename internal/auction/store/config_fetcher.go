package store

import (
	"context"
	"errors"
	"strconv"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/auction"
	"github.com/bidon-io/bidon-backend/internal/db"
	"gorm.io/gorm"
)

type ConfigFetcher struct {
	DB *db.DB
}

func (m *ConfigFetcher) Match(ctx context.Context, appID int64, adType ad.Type, segmentID int64) (*auction.Config, error) {
	dbConfig := &db.AuctionConfiguration{}

	query := m.DB.
		WithContext(ctx).
		Select("id", "public_uid", "external_win_notifications", "rounds").
		Where(map[string]any{
			"app_id":  appID,
			"ad_type": db.AdTypeFromDomain(adType),
		}).
		Order("created_at DESC")

	if segmentID != 0 {
		query = query.Where("segment_id = ? OR segment_id IS NULL", segmentID)
	} else {
		query = query.Where("segment_id IS NULL")
	}

	err := query.Take(dbConfig).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = auction.ErrNoAdsFound
		}
		return nil, err
	}

	config := &auction.Config{
		ID:                       dbConfig.ID,
		UID:                      strconv.FormatInt(dbConfig.PublicUID.Int64, 10),
		ExternalWinNotifications: *dbConfig.ExternalWinNotifications,
		Rounds:                   dbConfig.Rounds,
	}

	return config, nil
}

func (m *ConfigFetcher) FetchByUID(ctx context.Context, appID int64, key, id string) *auction.Config {
	if key != "id" && key != "uid" {
		return nil
	}

	dbConfig := &db.AuctionConfiguration{}

	filter := map[string]any{
		"app_id": appID,
	}
	if key == "id" {
		filter["id"] = id
	} else {
		filter["public_uid"] = id
	}

	err := m.DB.
		WithContext(ctx).
		Select("id", "public_uid", "external_win_notifications", "rounds").
		Where(filter).
		Order("created_at DESC").
		Take(dbConfig).
		Error
	if err != nil {
		return nil
	}

	config := &auction.Config{
		ID:                       dbConfig.ID,
		UID:                      strconv.FormatInt(dbConfig.PublicUID.Int64, 10),
		ExternalWinNotifications: *dbConfig.ExternalWinNotifications,
		Rounds:                   dbConfig.Rounds,
	}

	return config
}
