package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/db"
)

type tracingLogger struct {
	logger.Interface
}

func (l *tracingLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	span := trace.SpanFromContext(ctx)
	traceID := span.SpanContext().TraceID().String()

	sql, rows := fc()
	elapsed := time.Since(begin)

	log.Printf("[DEBUG_MIXUP] [TraceID: %s] SQL: %s | Rows: %d | Duration: %v | Error: %v",
		traceID, sql, rows, elapsed, err)
}

type ConfigurationFetcher struct {
	DB    *db.DB
	Cache cache
}

type cache interface {
	Get(context.Context, []byte, func(ctx context.Context) (adapter.RawConfigsMap, error)) (adapter.RawConfigsMap, error)
}

func (f *ConfigurationFetcher) FetchCached(ctx context.Context, appID int64, adapterKeys []adapter.Key) (adapter.RawConfigsMap, error) {
	// Sort adapter keys to get deterministic cache key
	key := f.cacheKey(appID, adapterKeys)
	if appID == 735528 {
		span := trace.SpanFromContext(ctx)
		log.Printf("[DEBUG_MIXUP] [TraceID: %s] FetchCached called for AppID %d with keys: %v. CacheKey: %x", span.SpanContext().TraceID(), appID, adapterKeys, key)
	}
	return f.Cache.Get(ctx, key, func(ctx context.Context) (adapter.RawConfigsMap, error) {
		if appID == 735528 {
			span := trace.SpanFromContext(ctx)
			log.Printf("[DEBUG_MIXUP] [TraceID: %s] Cache MISS for AppID %d. Fetching from DB...", span.SpanContext().TraceID(), appID)
		}
		return f.Fetch(ctx, appID, adapterKeys)
	})
}

func (f *ConfigurationFetcher) Fetch(ctx context.Context, appID int64, adapterKeys []adapter.Key) (adapter.RawConfigsMap, error) {
	var dbProfiles []db.AppDemandProfile

	tx := f.DB.WithContext(ctx)
	if appID == 735528 {
		span := trace.SpanFromContext(ctx)
		log.Printf("[DEBUG_MIXUP] [TraceID: %s] Fetch (DB) started for AppID %d", span.SpanContext().TraceID(), appID)
		// Use custom tracing logger to inject trace_id into SQL logs
		tx = tx.Session(&gorm.Session{Logger: &tracingLogger{Interface: logger.Default.LogMode(logger.Info)}})
	}

	err := tx.
		Select("app_demand_profiles.id, app_demand_profiles.account_id, app_demand_profiles.demand_source_id, app_demand_profiles.data").
		Where("app_id = ? AND app_demand_profiles.enabled = ?", appID, true).
		InnerJoins("Account", f.DB.Select("id", "extra")).
		InnerJoins("DemandSource", f.DB.Select("api_key").Where(map[string]any{"api_key": adapterKeys})).
		Find(&dbProfiles).
		Error
	if err != nil {
		return nil, fmt.Errorf("cannot load adapter config from DB: %w", err)
	}

	configs := adapter.RawConfigsMap{}
	for _, dbProfile := range dbProfiles {
		var extra map[string]any
		err = json.Unmarshal(dbProfile.Account.Extra, &extra)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal account extra: %v", err)
		}

		var data map[string]any
		err = json.Unmarshal(dbProfile.Data, &data)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal profile data: %v", err)
		}

		key := adapter.Key(dbProfile.DemandSource.APIKey)
		configs[key] = adapter.Config{
			AccountExtra: extra,
			AppData:      data,
		}
	}

	if appID == 735528 {
		span := trace.SpanFromContext(ctx)
		dbProfilesJSON, _ := json.Marshal(dbProfiles)
		log.Printf("[DEBUG_MIXUP] [TraceID: %s] DB Rows: %s", span.SpanContext().TraceID(), string(dbProfilesJSON))

		configsJSON, _ := json.Marshal(configs)
		log.Printf("[DEBUG_MIXUP] [TraceID: %s] Fetch returning configs: %s", span.SpanContext().TraceID(), string(configsJSON))
	}

	return configs, nil
}

func (f *ConfigurationFetcher) cacheKey(appID int64, adapterKeys []adapter.Key) []byte {
	stringKeys := make([]string, len(adapterKeys))
	for i, key := range adapterKeys {
		stringKeys[i] = string(key)
	}
	// Sort adapter keys to get deterministic cache key
	sort.Strings(stringKeys)
	key := fmt.Sprintf("%d:%s", appID, strings.Join(stringKeys, ":"))
	hash := sha256.Sum256([]byte(key))

	return hash[:]
}
