package admin

import (
	"context"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	v8n "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type DemandSourceAccount struct {
	ID int64 `json:"id"`
	DemandSourceAccountAttrs
	User         User         `json:"user"`
	DemandSource DemandSource `json:"demand_source"`
}

type DemandSourceAccountAttrs struct {
	UserID         int64          `json:"user_id"`
	Type           string         `json:"type"`
	DemandSourceID int64          `json:"demand_source_id"`
	IsBidding      *bool          `json:"is_bidding"`
	Extra          map[string]any `json:"extra"`
}

type DemandSourceAccountService = ResourceService[DemandSourceAccount, DemandSourceAccountAttrs]

func NewDemandSourceAccountService(store Store) *DemandSourceAccountService {
	s := &DemandSourceAccountService{
		repo: store.DemandSourceAccounts(),
	}

	s.policy = &demandSourceAccountPolicy{
		repo: store.DemandSourceAccounts(),
	}

	s.getValidator = func(attrs *DemandSourceAccountAttrs) v8n.ValidatableWithContext {
		return &demandSourceAccountValidator{
			attrs:            attrs,
			demandSourceRepo: store.DemandSources(),
		}
	}

	return s
}

//go:generate go run -mod=mod github.com/matryer/moq@latest -out demand_source_account_mocks_test.go . DemandSourceAccountRepo
type DemandSourceAccountRepo interface {
	ResourceRepo[DemandSourceAccount, DemandSourceAccountAttrs]

	ListOwnedByUserOrShared(ctx context.Context, userID int64) ([]DemandSourceAccount, error)
	FindOwnedByUserOrShared(ctx context.Context, userID int64, id int64) (*DemandSourceAccount, error)
}

type demandSourceAccountPolicy struct {
	repo DemandSourceAccountRepo
}

func (p *demandSourceAccountPolicy) scope(authCtx AuthContext) (resourceScope[DemandSourceAccount], error) {
	return &demandSourceAccountScope{
		repo:    p.repo,
		authCtx: authCtx,
	}, nil
}

type demandSourceAccountScope struct {
	repo DemandSourceAccountRepo

	authCtx AuthContext
}

func (s *demandSourceAccountScope) list(ctx context.Context) ([]DemandSourceAccount, error) {
	if s.authCtx.IsAdmin() {
		return s.repo.List(ctx)
	}

	return s.repo.ListOwnedByUserOrShared(ctx, s.authCtx.UserID())
}

func (s *demandSourceAccountScope) find(ctx context.Context, id int64) (*DemandSourceAccount, error) {
	if s.authCtx.IsAdmin() {
		return s.repo.Find(ctx, id)
	}

	return s.repo.FindOwnedByUserOrShared(ctx, s.authCtx.UserID(), id)
}

type demandSourceAccountValidator struct {
	attrs *DemandSourceAccountAttrs

	demandSourceRepo DemandSourceRepo
}

func (v *demandSourceAccountValidator) ValidateWithContext(ctx context.Context) error {
	demandSource, err := v.demandSourceRepo.Find(ctx, v.attrs.DemandSourceID)
	if err != nil {
		return v8n.NewInternalError(err)
	}

	return v8n.ValidateStruct(v.attrs,
		v8n.Field(&v.attrs.Extra, v.extraRule(demandSource)),
	)
}

func (v *demandSourceAccountValidator) extraRule(demandSource *DemandSource) v8n.Rule {
	var rule v8n.MapRule

	switch adapter.Key(demandSource.ApiKey) {
	case adapter.ApplovinKey:
		rule = v8n.Map(
			v8n.Key("sdk_key", v8n.Required, isString),
		)
	case adapter.BidmachineKey:
		rule = v8n.Map(
			v8n.Key("seller_id", v8n.Required, isString),
			v8n.Key("endpoint", v8n.Required, is.URL),
			v8n.Key("mediation_config", v8n.Required, v8n.Each(isString)),
		)
	case adapter.BigoAdsKey:
		rule = v8n.Map(
			v8n.Key("publisher_id", v8n.Required, isString),
			v8n.Key("endpoint", v8n.Required, is.URL),
		)
	case adapter.MintegralKey:
		rule = v8n.Map(
			v8n.Key("app_key", v8n.Required, isString),
			v8n.Key("publisher_id", v8n.Required, isString),
		)
	case adapter.VungleKey:
		rule = v8n.Map(
			v8n.Key("account_id", v8n.Required, isString),
		)
	}

	return rule.AllowExtraKeys()
}
