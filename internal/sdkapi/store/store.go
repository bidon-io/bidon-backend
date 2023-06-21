package store

import (
	"context"
	"errors"

	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"gorm.io/gorm"
)

type AppFetcher struct {
	DB *db.DB
}

func (f *AppFetcher) Fetch(ctx context.Context, appKey, appBundle string) (*sdkapi.App, error) {
	var dbApp db.App
	err := f.DB.
		WithContext(ctx).
		Select("id").
		Take(&dbApp, map[string]any{"app_key": appKey, "package_name": appBundle}).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = sdkapi.ErrAppNotValid
		}

		return nil, err
	}

	return &sdkapi.App{ID: dbApp.ID}, nil
}

type SegmentFetcher struct {
	DB *db.DB
}

func (f *SegmentFetcher) Fetch(ctx context.Context, appID int64) ([]db.Segment, error) {
	var dbSegments []db.Segment

	if err := f.DB.WithContext(ctx).Find(&dbSegments, map[string]any{"app_id": appID}).Error; err != nil {
		return nil, err
	}

	return dbSegments, nil
}
