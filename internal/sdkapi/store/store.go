package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"gorm.io/gorm"
)

type AppFetcher struct {
	DB    *db.DB
	Cache cache[sdkapi.App]
}

type cache[T any] interface {
	Get(context.Context, []byte, func(ctx context.Context) (T, error)) (T, error)
}

func (f *AppFetcher) FetchCached(ctx context.Context, appKey, appBundle string) (app sdkapi.App, err error) {
	cacheKey := fmt.Sprintf("app:%s:%s", appKey, appBundle)

	return f.Cache.Get(ctx, []byte(cacheKey), func(ctx context.Context) (sdkapi.App, error) {
		return f.Fetch(ctx, appKey, appBundle)
	})
}

func (f *AppFetcher) Fetch(ctx context.Context, appKey, appBundle string) (app sdkapi.App, err error) {
	var dbApp db.App
	err = f.DB.
		WithContext(ctx).
		Select("id").
		Take(&dbApp, map[string]any{"app_key": appKey, "package_name": appBundle}).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return app, sdkapi.ErrAppNotValid
		}

		return app, fmt.Errorf("fetch app: %v", err)
	}

	app.ID = dbApp.ID

	return app, nil
}

type AdapterInitConfigsFetcher struct {
	DB    *db.DB
	Cache cache[[]sdkapi.AdapterInitConfig]
}

func (f *AdapterInitConfigsFetcher) FetchAdapterInitConfigsCached(ctx context.Context, appID int64, adapterKeys []adapter.Key, setAmazonSlots bool, setOrder bool) ([]sdkapi.AdapterInitConfig, error) {
	key, err := f.cacheKey(appID, adapterKeys, setAmazonSlots, setOrder)
	if err != nil {
		return nil, err
	}
	return f.Cache.Get(ctx, key, func(ctx context.Context) ([]sdkapi.AdapterInitConfig, error) {
		return f.FetchAdapterInitConfigs(ctx, appID, adapterKeys, setAmazonSlots, setOrder)
	})
}

func (f *AdapterInitConfigsFetcher) FetchAdapterInitConfigs(ctx context.Context, appID int64, adapterKeys []adapter.Key, setAmazonSlots bool, setOrder bool) ([]sdkapi.AdapterInitConfig, error) {
	var dbProfiles []db.AppDemandProfile

	err := f.DB.
		WithContext(ctx).
		Select("app_demand_profiles.id, app_demand_profiles.data").
		Where("app_id", appID).
		InnerJoins("Account", f.DB.Select("id", "extra")).
		InnerJoins("Account.DemandSource", f.DB.Select("api_key").Where(map[string]any{"api_key": adapterKeys})).
		Find(&dbProfiles).
		Error
	if err != nil {
		return nil, fmt.Errorf("find app demand profiles: %v", err)
	}

	configs := make([]sdkapi.AdapterInitConfig, 0, len(dbProfiles))
	for _, profile := range dbProfiles {
		adapterKey := adapter.Key(profile.Account.DemandSource.APIKey)

		config, err := sdkapi.NewAdapterInitConfig(adapterKey, setOrder)
		if err != nil {
			return nil, fmt.Errorf("new AdapterInitConfig: %w", err)
		}

		err = json.Unmarshal(profile.Account.Extra, config)
		if err != nil {
			return nil, fmt.Errorf("unmarshal account extra: %v", err)
		}

		err = json.Unmarshal(profile.Data, config)
		if err != nil {
			return nil, fmt.Errorf("unmarshal profile data: %v", err)
		}

		applovinConfig, ok := config.(*sdkapi.ApplovinInitConfig)
		if ok {
			applovinConfig.AppKey = applovinConfig.SDKKey
		}

		if setAmazonSlots {
			amazonConfig, ok := config.(*sdkapi.AmazonInitConfig)
			if ok {
				amazonConfig.Slots, err = f.fetchAmazonSlots(ctx, appID)
				if err != nil {
					return nil, fmt.Errorf("fetch amazon slots: %v", err)
				}
			}
		}

		configs = append(configs, config)
	}

	return configs, nil
}

func (f *AdapterInitConfigsFetcher) fetchAmazonSlots(ctx context.Context, appID int64) ([]sdkapi.AmazonSlot, error) {
	var dbLineItems []db.LineItem

	err := f.DB.
		WithContext(ctx).
		Select("line_items.id, line_items.extra, line_items.ad_type, line_items.format").
		Where("app_id", appID).
		InnerJoins("Account", f.DB.Select("id")).
		InnerJoins("Account.DemandSource", f.DB.Select("api_key").Where("api_key", adapter.AmazonKey)).
		Order("line_items.id").
		Find(&dbLineItems).
		Error

	if err != nil {
		return nil, fmt.Errorf("find line items: %v", err)
	}

	slots := make([]sdkapi.AmazonSlot, 0, len(dbLineItems))
	for _, lineItem := range dbLineItems {
		slot := sdkapi.AmazonSlot{}

		slotUUID, ok := lineItem.Extra["slot_uuid"].(string)
		if !ok {
			return nil, fmt.Errorf("slot_uuid is either missing or not a string")
		}
		slot.SlotUUID = slotUUID

		format, ok := lineItem.Extra["format"].(string)
		if !ok {
			return nil, fmt.Errorf("format is either missing or not a string")
		}
		slot.Format = format

		slots = append(slots, slot)
	}

	return slots, nil
}

func (f *AdapterInitConfigsFetcher) cacheKey(appID int64, adapterKeys []adapter.Key, setAmazonSlots bool, setOrder bool) ([]byte, error) {
	// Sort adapter keys to get deterministic cache key
	sort.Slice(adapterKeys, func(i, j int) bool {
		return adapterKeys[i] < adapterKeys[j]
	})
	cacheKeyData := struct {
		AppID          int64         `json:"app_id"`
		AdapterKeys    []adapter.Key `json:"adapter_keys"`
		SetAmazonSlots bool          `json:"set_amazon_slots"`
		SetOrder       bool          `json:"set_order"`
	}{
		AppID:          appID,
		AdapterKeys:    adapterKeys,
		SetAmazonSlots: setAmazonSlots,
		SetOrder:       setOrder,
	}
	jsonData, err := json.Marshal(cacheKeyData)

	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(jsonData)
	return hash[:], nil
}
