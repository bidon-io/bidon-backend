package admin

import (
	"context"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	v8n "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

type LineItem struct {
	ID int64 `json:"id"`
	LineItemAttrs
	App     App                 `json:"app"`
	Account DemandSourceAccount `json:"account"`
}

type LineItemAttrs struct {
	HumanName   string           `json:"human_name"`
	AppID       int64            `json:"app_id"`
	BidFloor    *decimal.Decimal `json:"bid_floor"`
	AdType      ad.Type          `json:"ad_type"`
	Format      *ad.Format       `json:"format"`
	AccountID   int64            `json:"account_id"`
	AccountType string           `json:"account_type"`
	Code        *string          `json:"code"`
	Extra       map[string]any   `json:"extra"`
}

type LineItemService = ResourceService[LineItem, LineItemAttrs]

func NewLineItemService(store Store) *LineItemService {
	s := &LineItemService{
		repo: store.LineItems(),
	}

	s.policy = &lineItemPolicy{
		repo: store.LineItems(),
	}

	s.getValidator = func(attrs *LineItemAttrs) v8n.ValidatableWithContext {
		return &lineItemAttrsValidator{
			attrs:                   attrs,
			demandSourceAccountRepo: store.DemandSourceAccounts(),
		}
	}

	return s
}

type LineItemRepo interface {
	ResourceRepo[LineItem, LineItemAttrs]

	ListOwnedByUser(ctx context.Context, userID int64) ([]LineItem, error)
	FindOwnedByUser(ctx context.Context, userID, id int64) (*LineItem, error)
}

type lineItemPolicy struct {
	repo LineItemRepo
}

func (p *lineItemPolicy) scope(authCtx AuthContext) (resourceScope[LineItem], error) {
	return &lineItemScope{
		repo:    p.repo,
		authCtx: authCtx,
	}, nil
}

type lineItemScope struct {
	repo LineItemRepo

	authCtx AuthContext
}

func (s *lineItemScope) list(ctx context.Context) ([]LineItem, error) {
	if s.authCtx.IsAdmin() {
		return s.repo.List(ctx)
	}

	return s.repo.ListOwnedByUser(ctx, s.authCtx.UserID())
}

func (s *lineItemScope) find(ctx context.Context, id int64) (*LineItem, error) {
	if s.authCtx.IsAdmin() {
		return s.repo.Find(ctx, id)
	}

	return s.repo.FindOwnedByUser(ctx, s.authCtx.UserID(), id)
}

type lineItemAttrsValidator struct {
	attrs *LineItemAttrs

	demandSourceAccountRepo DemandSourceAccountRepo
}

func (v *lineItemAttrsValidator) ValidateWithContext(ctx context.Context) error {
	account, err := v.demandSourceAccountRepo.Find(ctx, v.attrs.AccountID)
	if err != nil {
		return v8n.NewInternalError(err)
	}

	return v8n.ValidateStruct(v.attrs,
		v8n.Field(&v.attrs.Extra, v.extraRule(account)),
	)
}

func (v *lineItemAttrsValidator) extraRule(account *DemandSourceAccount) v8n.Rule {
	var rule v8n.MapRule

	switch adapter.Key(account.DemandSource.ApiKey) {
	case adapter.AdmobKey:
		rule = v8n.Map(
			v8n.Key("ad_unit_id", v8n.Required, isString),
		)
	case adapter.ApplovinKey:
		rule = v8n.Map(
			v8n.Key("zone_id", v8n.Required, isString),
		)
	case adapter.BigoAdsKey:
		rule = v8n.Map(
			v8n.Key("slot_id", v8n.Required, isString),
		)
	case adapter.DTExchangeKey, adapter.MetaKey, adapter.UnityAdsKey, adapter.VungleKey, adapter.MobileFuseKey:
		rule = v8n.Map(
			v8n.Key("placement_id", v8n.Required, isString),
		)
	case adapter.MintegralKey:
		rule = v8n.Map(
			v8n.Key("placement_id", v8n.Required, isString),
			v8n.Key("ad_unit_id", v8n.Required, isString),
		)
	}

	return rule.AllowExtraKeys()
}
