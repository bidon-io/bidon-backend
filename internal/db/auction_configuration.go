package db

import (
	"database/sql"
	"errors"
	"gorm.io/gorm"
)

func (ac *AuctionConfiguration) BeforeSave(tx *gorm.DB) (err error) {
	// Check if the combination of app_id, ad_type, and segment_id is already taken
	var count int64

	query := tx.Model(&AuctionConfiguration{}).
		Where("app_id = ? AND ad_type = ?", ac.AppID, ac.AdType)

	if ac.SegmentID != nil && ac.SegmentID.Valid {
		query = query.Where("segment_id = ?", ac.SegmentID.Int64)
	} else {
		query = query.Where("segment_id IS NULL")
	}

	query = query.Not(ac.Model.ID).Count(&count)

	if query.Error != nil {
		return query.Error
	}

	if count > 0 {
		return errors.New("the combination of app_id, ad_type, and segment_id already exists")
	}

	return nil
}

// BeforeCreate hook to set PublicUID
func (ac *AuctionConfiguration) BeforeCreate(tx *gorm.DB) (err error) {
	publicUID, err := generatePublicUID(tx)
	if err != nil {
		return err
	}

	ac.PublicUID = &sql.NullInt64{Int64: publicUID, Valid: true}
	return nil
}
