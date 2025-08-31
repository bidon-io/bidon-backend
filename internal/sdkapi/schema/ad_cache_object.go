package schema

import "github.com/shopspring/decimal"

type AdCacheObject struct {
	DemandID  string          `json:"demand_id" validate:"required"`
	Timestamp int64           `json:"timestamp" validate:"required"`
	Price     decimal.Decimal `json:"price" validate:"required"`
	AdUnitUID *string         `json:"ad_unit_uid,omitempty"`
}
