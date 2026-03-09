package adminstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/admin"
	"github.com/bidon-io/bidon-backend/internal/admin/resource"
	"github.com/bidon-io/bidon-backend/internal/audit"
	"github.com/bidon-io/bidon-backend/internal/db"
)

type LineItemRepo struct {
	*resourceRepo[admin.LineItem, admin.LineItemAttrs, db.LineItem]
}

func NewLineItemRepo(d *db.DB) *LineItemRepo {
	return &LineItemRepo{
		resourceRepo: &resourceRepo[admin.LineItem, admin.LineItemAttrs, db.LineItem]{
			db:           d,
			mapper:       lineItemMapper{db: d},
			associations: []string{"App", "Account"},
		},
	}
}

func (r *LineItemRepo) List(ctx context.Context, qParams map[string][]string) (*resource.Collection[admin.LineItem], error) {
	filters := queryToLineItemFilters(qParams)
	pgn := PaginationFromQueryParams[db.LineItem](qParams)
	return r.list(ctx, filters.apply, pgn)
}

func (r *LineItemRepo) ListOwnedByUser(ctx context.Context, userID int64, qParams map[string][]string) (*resource.Collection[admin.LineItem], error) {
	filters := queryToLineItemFilters(qParams)
	filters.UserID = userID
	pgn := PaginationFromQueryParams[db.LineItem](qParams)
	return r.list(ctx, filters.apply, pgn)
}

func (r *LineItemRepo) FindOwnedByUser(ctx context.Context, userID int64, id int64) (*admin.LineItem, error) {
	return r.find(ctx, id, func(db *gorm.DB) *gorm.DB {
		s := db.Session(&gorm.Session{NewDB: true})
		return db.InnerJoins("App", s.Table("App").Where(map[string]any{"user_id": userID}))
	})
}

func (r *LineItemRepo) Create(ctx context.Context, attrs *admin.LineItemAttrs) (*admin.LineItem, error) {
	dbItem := r.mapper.dbModel(attrs, 0)

	id, alreadyExists, err := r.firstOrCreate(ctx, dbItem)
	if err != nil {
		return nil, err
	}

	item, err := r.Find(ctx, id)
	if err != nil {
		return nil, err
	}

	item.AlreadyExists = alreadyExists

	return item, nil
}

func (r *LineItemRepo) firstOrCreate(ctx context.Context, item *db.LineItem) (id int64, exists bool, err error) {
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := audit.SetContext(tx, ctx); err != nil {
			return err
		}

		existingID, err := r.findExistingID(tx, item)
		if err != nil {
			return err
		}

		if existingID != 0 {
			id = existingID
			exists = true
		} else {
			if err := tx.Create(item).Error; err != nil {
				return err
			}
			id = item.ID
			exists = false
		}

		return nil
	})

	if err != nil {
		return 0, false, err
	}

	return id, exists, nil
}

func (r *LineItemRepo) CreateMany(ctx context.Context, items []admin.LineItemAttrs) error {
	dbItems := make([]*db.LineItem, len(items))
	for i := range items {
		dbItems[i] = r.mapper.dbModel(&items[i], 0)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := audit.SetContext(tx, ctx); err != nil {
			return err
		}
		return tx.Create(&dbItems).Error
	})
}

func (r *LineItemRepo) findExistingID(tx *gorm.DB, item *db.LineItem) (int64, error) {
	query := tx.Model(&db.LineItem{}).
		Select("id").
		Where("app_id = ?", item.AppID).
		Where("account_type = ?", item.AccountType).
		Where("account_id = ?", item.AccountID).
		Where("human_name = ?", item.HumanName).
		Where("ad_type = ?", item.AdType)

	if item.BidFloor.Valid {
		query = query.Where("bid_floor = ?", item.BidFloor.Decimal)
	} else {
		query = query.Where("bid_floor IS NULL")
	}

	if item.Format.Valid {
		query = query.Where("format = ?", item.Format.String)
	} else {
		query = query.Where("format IS NULL")
	}

	if item.IsBidding.Valid {
		query = query.Where("bidding = ?", item.IsBidding.Bool)
	} else {
		query = query.Where("bidding IS NULL")
	}

	extraJSON, err := json.Marshal(item.Extra)
	if err != nil {
		return 0, err
	}
	if string(extraJSON) == "null" {
		query = query.Where("extra IS NULL OR extra = '{}'::jsonb")
	} else {
		query = query.Where("extra = ?::jsonb", string(extraJSON))
	}

	var existing db.LineItem
	if err := query.Take(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}

	return existing.ID, nil
}

type lineItemMapper struct {
	db *db.DB
}

//lint:ignore U1000 this method is used by generic struct
func (m lineItemMapper) dbModel(i *admin.LineItemAttrs, id int64) *db.LineItem {
	bidFloor := decimal.NullDecimal{}
	if i.BidFloor != nil {
		bidFloor.Decimal = *i.BidFloor
		bidFloor.Valid = true
	}

	format := sql.NullString{}
	if i.Format != nil {
		format.String = string(*i.Format)
		format.Valid = true
	}

	var isBidding sql.NullBool
	if i.IsBidding != nil {
		isBidding.Bool = *i.IsBidding
		isBidding.Valid = true
	}

	return &db.LineItem{
		ID:          id,
		AppID:       i.AppID,
		AccountType: i.AccountType,
		AccountID:   i.AccountID,
		HumanName:   i.HumanName,
		BidFloor:    bidFloor,
		AdType:      db.AdTypeFromDomain(i.AdType),
		Extra:       i.Extra,
		Format:      format,
		IsBidding:   isBidding,
	}
}

//lint:ignore U1000 this method is used by generic struct
func (m lineItemMapper) resource(i *db.LineItem) admin.LineItem {
	return admin.LineItem{
		ID:            i.ID,
		PublicUID:     strconv.FormatInt(i.PublicUID.Int64, 10),
		LineItemAttrs: m.resourceAttrs(i),
		App: admin.App{
			ID:       i.AppID,
			AppAttrs: appMapper{}.resourceAttrs(&i.App),
		},
		Account: admin.DemandSourceAccount{
			ID:                       i.AccountID,
			DemandSourceAccountAttrs: demandSourceAccountMapper{}.resourceAttrs(&i.Account),
		},
	}
}

func (m lineItemMapper) resourceAttrs(i *db.LineItem) admin.LineItemAttrs {
	format := ad.Format(i.Format.String)
	return admin.LineItemAttrs{
		HumanName:   i.HumanName,
		AppID:       i.AppID,
		BidFloor:    &i.BidFloor.Decimal,
		AdType:      i.AdType.Domain(),
		Format:      &format,
		AccountID:   i.AccountID,
		AccountType: i.AccountType,
		IsBidding:   &i.IsBidding.Bool,
		Extra:       i.Extra,
	}
}

type lineItemFilters struct {
	UserID      int64
	AppID       int64
	AdType      db.AdType
	Format      string
	AccountID   int64
	AccountType string
	IsBidding   *bool
	HumanName   string
}

func (f *lineItemFilters) apply(db *gorm.DB) *gorm.DB {
	if f.UserID != 0 {
		db = db.Joins("INNER JOIN apps ON apps.id = line_items.app_id").Where("apps.user_id = ?", f.UserID)
	}
	if f.AppID != 0 {
		db = db.Where("app_id = ?", f.AppID)
	}
	if f.AdType != 0 {
		db = db.Where("ad_type = ?", f.AdType)
	}
	if f.Format != "" {
		db = db.Where("format = ?", f.Format)
	}
	if f.AccountID != 0 {
		db = db.Where("account_id = ?", f.AccountID)
	}
	if f.AccountType != "" {
		db = db.Where("account_type = ?", f.AccountType)
	}
	if f.IsBidding != nil {
		if *f.IsBidding {
			db = db.Where("bidding = ?", true)
		} else {
			db = db.Where("bidding = ? OR bidding IS NULL", false)
		}
	}
	if f.HumanName != "" {
		db = db.Where("human_name ILIKE ?", "%"+f.HumanName+"%")
	}
	return db
}

func queryToLineItemFilters(qParams map[string][]string) lineItemFilters {
	filters := lineItemFilters{}
	if v, ok := qParams["user_id"]; ok {
		filters.UserID, _ = strconv.ParseInt(v[0], 10, 64)
	}
	if v, ok := qParams["app_id"]; ok {
		filters.AppID, _ = strconv.ParseInt(v[0], 10, 64)
	}
	if v, ok := qParams["ad_type"]; ok {
		dbAdType := db.AdTypeFromDomain(ad.Type(v[0]))
		filters.AdType = dbAdType
	}
	if v, ok := qParams["format"]; ok {
		filters.Format = v[0]
	}
	if v, ok := qParams["account_id"]; ok {
		filters.AccountID, _ = strconv.ParseInt(v[0], 10, 64)
	}
	if v, ok := qParams["account_type"]; ok {
		filters.AccountType = v[0]
	}
	if v, ok := qParams["is_bidding"]; ok {
		b := v[0] == "true"
		filters.IsBidding = &b
	}
	if v, ok := qParams["human_name"]; ok {
		filters.HumanName = v[0]
	}
	return filters
}
