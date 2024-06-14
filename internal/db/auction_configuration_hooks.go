package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

func (ac *AuctionConfiguration) BeforeSave(tx *gorm.DB) (err error) {
	// Check if the combination of app_id, ad_type, and segment_id is already taken
	var count int64

	query := tx.Model(&AuctionConfiguration{}).
		Where("app_id = ? AND ad_type = ?", ac.AppID, ac.AdType)

	var settings map[string]any
	if ac.Settings != nil {
		err = json.Unmarshal(ac.Settings, &settings)
		if err != nil {
			return fmt.Errorf("unmarshal settings: %v", err)
		}
	}

	if isV2, ok := settings["v2"].(bool); ok && isV2 {
		query = query.Where("settings->>'v2' = 'true'")
	} else {
		query = query.Where("settings->>'v2' IS NULL")
	}

	if ac.SegmentID != nil && ac.SegmentID.Valid {
		query = query.Where("segment_id = ?", ac.SegmentID.Int64)
	} else {
		query = query.Where("segment_id IS NULL")
	}

	query = query.Not(ac.ID).Count(&count)

	if query.Error != nil {
		return query.Error
	}

	if count > 0 {
		return errors.New("the combination of app_id, ad_type, and segment_id already exists")
	}

	return nil
}

func (ac *AuctionConfiguration) BeforeCreate(tx *gorm.DB) error {
	if ac.PublicUID == (sql.NullInt64{}) {
		id, err := generateSnowflakeID(tx)
		if err != nil {
			return fmt.Errorf("generate snowflake id: %v", err)
		}

		ac.PublicUID = sql.NullInt64{
			Int64: id,
			Valid: true,
		}
	}

	return nil
}
