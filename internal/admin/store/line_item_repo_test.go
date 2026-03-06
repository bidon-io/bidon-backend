package adminstore_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/admin"
	"github.com/bidon-io/bidon-backend/internal/admin/resource"
	adminstore "github.com/bidon-io/bidon-backend/internal/admin/store"
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/db/dbtest"
)

func TestLineItemRepo_List(t *testing.T) {
	tx := testDB.Begin()
	defer tx.Rollback()

	repo := adminstore.NewLineItemRepo(tx)

	users := make([]db.User, 2)
	for i := range users {
		users[i] = dbtest.CreateUser(t, tx)
	}
	apps := make([]db.App, 2)
	for i := range apps {
		apps[i] = dbtest.CreateApp(t, tx, func(app *db.App) {
			app.User = users[i]
		})
	}

	applovinDemandSource := dbtest.CreateDemandSource(t, tx, func(source *db.DemandSource) {
		source.APIKey = string(adapter.ApplovinKey)
		source.HumanName = source.APIKey
	})
	applovinAccount := dbtest.CreateDemandSourceAccount(t, tx, func(account *db.DemandSourceAccount) {
		account.User = users[0]
		account.DemandSource = applovinDemandSource
	})

	bidmachineDemandSource := dbtest.CreateDemandSource(t, tx, func(source *db.DemandSource) {
		source.APIKey = string(adapter.BidmachineKey)
		source.HumanName = source.APIKey
	})
	bidmachineAccount := dbtest.CreateDemandSourceAccount(t, tx, func(account *db.DemandSourceAccount) {
		account.User = users[0]
		account.DemandSource = bidmachineDemandSource
	})

	unityAdsDemandSource := dbtest.CreateDemandSource(t, tx, func(source *db.DemandSource) {
		source.APIKey = string(adapter.UnityAdsKey)
		source.HumanName = source.APIKey
	})
	unityAdsAccount1 := dbtest.CreateDemandSourceAccount(t, tx, func(account *db.DemandSourceAccount) {
		account.User = users[0]
		account.DemandSource = unityAdsDemandSource
	})
	unityAdsAccount2 := dbtest.CreateDemandSourceAccount(t, tx, func(account *db.DemandSourceAccount) {
		account.User = users[1]
		account.DemandSource = unityAdsDemandSource
	})

	items := []struct {
		*admin.LineItemAttrs
		App     db.App
		Account db.DemandSourceAccount
	}{
		{
			LineItemAttrs: &admin.LineItemAttrs{
				HumanName:   "banner",
				AppID:       apps[0].ID,
				BidFloor:    ptr(decimal.NewFromInt(1)),
				AdType:      ad.BannerType,
				Format:      ptr(ad.BannerFormat),
				AccountID:   applovinAccount.ID,
				AccountType: applovinAccount.Type,
				Extra:       map[string]any{"key": "value"},
			},
			App:     apps[0],
			Account: applovinAccount,
		},
		{
			LineItemAttrs: &admin.LineItemAttrs{
				HumanName:   "interstitial",
				AppID:       apps[0].ID,
				BidFloor:    ptr(decimal.Decimal{}),
				AdType:      ad.InterstitialType,
				Format:      ptr(ad.EmptyFormat),
				AccountID:   bidmachineAccount.ID,
				AccountType: bidmachineAccount.Type,
				Extra:       map[string]any{"key": "value"},
			},
			App:     apps[0],
			Account: bidmachineAccount,
		},
		{
			LineItemAttrs: &admin.LineItemAttrs{
				HumanName:   "rewarded",
				AppID:       apps[0].ID,
				BidFloor:    ptr(decimal.NewFromInt(3)),
				AdType:      ad.RewardedType,
				Format:      ptr(ad.EmptyFormat),
				AccountID:   unityAdsAccount1.ID,
				AccountType: unityAdsAccount1.Type,
				Extra:       map[string]any{"key": "value"},
			},
			App:     apps[0],
			Account: unityAdsAccount1,
		},
		{
			LineItemAttrs: &admin.LineItemAttrs{
				HumanName:   "rewarded App 2",
				AppID:       apps[1].ID,
				BidFloor:    ptr(decimal.NewFromInt(3)),
				AdType:      ad.RewardedType,
				Format:      ptr(ad.EmptyFormat),
				AccountID:   unityAdsAccount2.ID,
				AccountType: unityAdsAccount2.Type,
				IsBidding:   ptr(true),
				Extra:       map[string]any{"key": "value"},
			},
			App:     apps[1],
			Account: unityAdsAccount2,
		},
	}

	allItems := make([]admin.LineItem, len(items))
	for i, attrs := range items {
		item, err := repo.Create(context.Background(), attrs.LineItemAttrs)
		if err != nil {
			t.Fatalf("repo.Create(ctx, %+v) = %v, %q; allItems %T, %v", &attrs, nil, err, item, nil)
		}

		allItems[i] = *item
		allItems[i].Account = adminstore.DemandSourceAccountAttrsWithId(&attrs.Account)
		allItems[i].App = adminstore.AppAttrsWithId(&attrs.App)
	}

	testcases := []struct {
		name    string
		qParams map[string][]string
		want    *resource.Collection[admin.LineItem]
		wantErr bool
	}{
		{
			name:    "no filters",
			qParams: nil,
			want: &resource.Collection[admin.LineItem]{
				Items: allItems,
				Meta:  resource.CollectionMeta{TotalCount: 4},
			},
		},
		{
			name: "filter by user_id",
			qParams: map[string][]string{
				"user_id": {fmt.Sprint(users[0].ID)},
			},
			want: &resource.Collection[admin.LineItem]{
				Items: allItems[:3],
				Meta:  resource.CollectionMeta{TotalCount: 3},
			},
		},
		{
			name: "filter by app_id",
			qParams: map[string][]string{
				"app_id": {fmt.Sprint(apps[0].ID)},
			},
			want: &resource.Collection[admin.LineItem]{
				Items: allItems[:3],
				Meta:  resource.CollectionMeta{TotalCount: 3},
			},
		},
		{
			name: "filter by ad_type",
			qParams: map[string][]string{
				"ad_type": {string(ad.RewardedType)},
			},
			want: &resource.Collection[admin.LineItem]{
				Items: allItems[2:],
				Meta:  resource.CollectionMeta{TotalCount: 2},
			},
		},
		{
			name: "filter by account_id",
			qParams: map[string][]string{
				"account_id": {fmt.Sprint(unityAdsAccount1.ID)},
			},
			want: &resource.Collection[admin.LineItem]{
				Items: allItems[2:3],
				Meta:  resource.CollectionMeta{TotalCount: 1},
			},
		},
		{
			name: "filter by account_type",
			qParams: map[string][]string{
				"account_type": {unityAdsAccount1.Type},
			},
			want: &resource.Collection[admin.LineItem]{
				Items: allItems[2:],
				Meta:  resource.CollectionMeta{TotalCount: 2},
			},
		},
		{
			name: "filter by is_bidding true",
			qParams: map[string][]string{
				"is_bidding": {"true"},
			},
			want: &resource.Collection[admin.LineItem]{
				Items: allItems[3:],
				Meta:  resource.CollectionMeta{TotalCount: 1},
			},
		},
		{
			name: "filter by is_bidding false",
			qParams: map[string][]string{
				"is_bidding": {"false"},
			},
			want: &resource.Collection[admin.LineItem]{
				Items: allItems[:3],
				Meta:  resource.CollectionMeta{TotalCount: 3},
			},
		},
		{
			name: "filter by AppID and AccountID",
			qParams: map[string][]string{
				"app_id":     {fmt.Sprint(apps[0].ID)},
				"account_id": {fmt.Sprint(applovinAccount.ID)},
			},
			want: &resource.Collection[admin.LineItem]{
				Items: allItems[:1],
				Meta:  resource.CollectionMeta{TotalCount: 1},
			},
		},
		{
			name: "with pagination",
			qParams: map[string][]string{
				"limit": {"2"},
				"page":  {"2"},
			},
			want: &resource.Collection[admin.LineItem]{
				Items: allItems[2:],
				Meta:  resource.CollectionMeta{TotalCount: 4},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.List(context.Background(), tc.qParams)
			if err != nil {
				t.Fatalf("repo.List(ctx) = %v, %q; want %+v, %v", got, err, tc.want, nil)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("repo.List(ctx) mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestLineItemRepo_Find(t *testing.T) {
	tx := testDB.Begin()
	defer tx.Rollback()

	repo := adminstore.NewLineItemRepo(tx)

	app := dbtest.CreateApp(t, tx)
	applovinDemandSource := dbtest.CreateDemandSource(t, tx, func(source *db.DemandSource) {
		source.APIKey = string(adapter.ApplovinKey)
		source.HumanName = source.APIKey
	})
	applovinAccount := dbtest.CreateDemandSourceAccount(t, tx, func(account *db.DemandSourceAccount) {
		account.DemandSource = applovinDemandSource
	})

	attrs := &admin.LineItemAttrs{
		HumanName:   "banner",
		AppID:       app.ID,
		BidFloor:    ptr(decimal.NewFromInt(1)),
		AdType:      ad.BannerType,
		Format:      ptr(ad.BannerFormat),
		AccountID:   applovinAccount.ID,
		AccountType: applovinAccount.Type,
		Extra:       map[string]any{"key": "value"},
	}

	want, err := repo.Create(context.Background(), attrs)
	if err != nil {
		t.Fatalf("repo.Create(ctx, %+v) = %v, %q; want %T, %v", attrs, nil, err, want, nil)
	}
	want.App = adminstore.AppAttrsWithId(&app)
	want.Account = adminstore.DemandSourceAccountAttrsWithId(&applovinAccount)

	got, err := repo.Find(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("repo.Find(ctx) = %v, %q; want %+v, %v", got, err, want, nil)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("repo.List(ctx) mismatch (-want, +got):\n%s", diff)
	}
}

func TestLineItemRepo_Create_ReturnsExistingWhenAttrsMatch(t *testing.T) {
	tx := testDB.Begin()
	defer tx.Rollback()

	repo := adminstore.NewLineItemRepo(tx)

	app := dbtest.CreateApp(t, tx)
	applovinDemandSource := dbtest.CreateDemandSource(t, tx, func(source *db.DemandSource) {
		source.APIKey = string(adapter.ApplovinKey)
		source.HumanName = source.APIKey
	})
	applovinAccount := dbtest.CreateDemandSourceAccount(t, tx, func(account *db.DemandSourceAccount) {
		account.DemandSource = applovinDemandSource
	})

	attrs := &admin.LineItemAttrs{
		HumanName:   "banner",
		AppID:       app.ID,
		BidFloor:    ptr(decimal.NewFromInt(1)),
		AdType:      ad.BannerType,
		Format:      ptr(ad.BannerFormat),
		AccountID:   applovinAccount.ID,
		AccountType: applovinAccount.Type,
		IsBidding:   ptr(true),
		Extra: map[string]any{
			"placement_id": "abc",
			"mediation":    "max",
		},
	}

	first, err := repo.Create(context.Background(), attrs)
	if err != nil {
		t.Fatalf("first repo.Create(ctx, %+v) = %v, %q", attrs, first, err)
	}

	second, err := repo.Create(context.Background(), attrs)
	if err != nil {
		t.Fatalf("second repo.Create(ctx, %+v) = %v, %q", attrs, second, err)
	}

	if first.ID != second.ID {
		t.Fatalf("repo.Create(ctx, %+v) created duplicate records: first id %d, second id %d", attrs, first.ID, second.ID)
	}

	got, err := repo.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("repo.List(ctx) = %v, %q", got, err)
	}

	if got.Meta.TotalCount != 1 {
		t.Fatalf("repo.List(ctx) total count = %d, want 1", got.Meta.TotalCount)
	}
}

func TestLineItemRepo_Update(t *testing.T) {
	tx := testDB.Begin()
	defer tx.Rollback()

	repo := adminstore.NewLineItemRepo(tx)

	app := dbtest.CreateApp(t, tx)
	applovinDemandSource := dbtest.CreateDemandSource(t, tx, func(source *db.DemandSource) {
		source.APIKey = string(adapter.ApplovinKey)
		source.HumanName = source.APIKey
	})
	applovinAccount := dbtest.CreateDemandSourceAccount(t, tx, func(account *db.DemandSourceAccount) {
		account.DemandSource = applovinDemandSource
	})

	attrs := admin.LineItemAttrs{
		HumanName:   "banner",
		AppID:       app.ID,
		BidFloor:    ptr(decimal.NewFromInt(1)),
		AdType:      ad.BannerType,
		Format:      ptr(ad.BannerFormat),
		AccountID:   applovinAccount.ID,
		AccountType: applovinAccount.Type,
		Extra:       map[string]any{"key": "value"},
	}

	item, err := repo.Create(context.Background(), &attrs)
	if err != nil {
		t.Fatalf("repo.Create(ctx, %+v) = %v, %q; want %T, %v", &attrs, nil, err, item, nil)
	}

	want := item
	want.BidFloor = ptr(decimal.Decimal{})
	want.Format = ptr(ad.EmptyFormat)

	updateParams := &admin.LineItemAttrs{
		BidFloor: want.BidFloor,
		Format:   want.Format,
	}
	got, err := repo.Update(context.Background(), item.ID, updateParams)
	if err != nil {
		t.Fatalf("repo.Update(ctx, %+v) = %v, %q; want %T, %v", updateParams, nil, err, got, nil)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("repo.Find(ctx) mismatch (-want, +got):\n%s", diff)
	}
}

func TestLineItemRepo_Delete(t *testing.T) {
	tx := testDB.Begin()
	defer tx.Rollback()

	repo := adminstore.NewLineItemRepo(tx)

	app := dbtest.CreateApp(t, tx)
	applovinDemandSource := dbtest.CreateDemandSource(t, tx, func(source *db.DemandSource) {
		source.APIKey = string(adapter.ApplovinKey)
		source.HumanName = source.APIKey
	})
	applovinAccount := dbtest.CreateDemandSourceAccount(t, tx, func(account *db.DemandSourceAccount) {
		account.DemandSource = applovinDemandSource
	})
	attrs := &admin.LineItemAttrs{
		HumanName:   "banner",
		AppID:       app.ID,
		BidFloor:    ptr(decimal.NewFromInt(1)),
		AdType:      ad.BannerType,
		Format:      ptr(ad.BannerFormat),
		AccountID:   applovinAccount.ID,
		AccountType: applovinAccount.Type,
		Extra:       map[string]any{"key": "value"},
	}
	item, err := repo.Create(context.Background(), attrs)
	if err != nil {
		t.Fatalf("repo.Create(ctx, %+v) = %v, %q; want %T, %v", attrs, nil, err, item, nil)
	}

	err = repo.Delete(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("repo.Delete(ctx, %v) = %q, want %v", item.ID, err, nil)
	}

	got, err := repo.Find(context.Background(), item.ID)
	if got != nil {
		t.Fatalf("repo.Find(ctx, %v) = %+v, %q; want %v, %q", item.ID, got, err, nil, "record not found")
	}
}

func TestLineItemRepo_ListOwnedByUser(t *testing.T) {
	tx := testDB.Begin()
	defer tx.Rollback()

	repo := adminstore.NewLineItemRepo(tx)

	owner := dbtest.CreateUser(t, tx)
	anotherUser := dbtest.CreateUser(t, tx)
	ownerApp := dbtest.CreateApp(t, tx, func(app *db.App) {
		app.User = owner
	})
	anotherApp := dbtest.CreateApp(t, tx, func(app *db.App) {
		app.User = anotherUser
	})
	demandSource := dbtest.CreateDemandSource(t, tx, func(source *db.DemandSource) {
		source.APIKey = string(adapter.ApplovinKey)
		source.HumanName = source.APIKey
	})
	account := dbtest.CreateDemandSourceAccount(t, tx, func(acc *db.DemandSourceAccount) {
		acc.User = owner
		acc.DemandSource = demandSource
	})

	ownerItem, err := repo.Create(context.Background(), &admin.LineItemAttrs{
		HumanName:   "owner-item",
		AppID:       ownerApp.ID,
		BidFloor:    ptr(decimal.NewFromFloat(0.1)),
		AdType:      ad.BannerType,
		Format:      ptr(ad.BannerFormat),
		AccountID:   account.ID,
		AccountType: account.Type,
		Extra:       map[string]any{"ad_unit_id": "owner"},
	})
	if err != nil {
		t.Fatalf("repo.Create(owner) error: %v", err)
	}

	_, err = repo.Create(context.Background(), &admin.LineItemAttrs{
		HumanName:   "another-item",
		AppID:       anotherApp.ID,
		BidFloor:    ptr(decimal.NewFromFloat(0.2)),
		AdType:      ad.BannerType,
		Format:      ptr(ad.BannerFormat),
		AccountID:   account.ID,
		AccountType: account.Type,
		Extra:       map[string]any{"ad_unit_id": "another"},
	})
	if err != nil {
		t.Fatalf("repo.Create(another) error: %v", err)
	}

	got, err := repo.ListOwnedByUser(context.Background(), owner.ID, nil)
	if err != nil {
		t.Fatalf("repo.ListOwnedByUser() error: %v", err)
	}
	if got.Meta.TotalCount != 1 {
		t.Fatalf("repo.ListOwnedByUser() total count = %d, want 1", got.Meta.TotalCount)
	}
	if len(got.Items) != 1 || got.Items[0].ID != ownerItem.ID {
		t.Fatalf("repo.ListOwnedByUser() returned wrong items: %+v", got.Items)
	}
}

func TestLineItemRepo_FindOwnedByUser(t *testing.T) {
	tx := testDB.Begin()
	defer tx.Rollback()

	repo := adminstore.NewLineItemRepo(tx)

	owner := dbtest.CreateUser(t, tx)
	anotherUser := dbtest.CreateUser(t, tx)
	ownerApp := dbtest.CreateApp(t, tx, func(app *db.App) {
		app.User = owner
	})
	demandSource := dbtest.CreateDemandSource(t, tx, func(source *db.DemandSource) {
		source.APIKey = string(adapter.ApplovinKey)
		source.HumanName = source.APIKey
	})
	account := dbtest.CreateDemandSourceAccount(t, tx, func(acc *db.DemandSourceAccount) {
		acc.User = owner
		acc.DemandSource = demandSource
	})

	item, err := repo.Create(context.Background(), &admin.LineItemAttrs{
		HumanName:   "owner-item",
		AppID:       ownerApp.ID,
		BidFloor:    ptr(decimal.NewFromFloat(0.1)),
		AdType:      ad.BannerType,
		Format:      ptr(ad.BannerFormat),
		AccountID:   account.ID,
		AccountType: account.Type,
		Extra:       map[string]any{"ad_unit_id": "owner"},
	})
	if err != nil {
		t.Fatalf("repo.Create() error: %v", err)
	}

	got, err := repo.FindOwnedByUser(context.Background(), owner.ID, item.ID)
	if err != nil {
		t.Fatalf("repo.FindOwnedByUser(owner) error: %v", err)
	}
	if got.ID != item.ID {
		t.Fatalf("repo.FindOwnedByUser(owner) ID = %d, want %d", got.ID, item.ID)
	}

	got, err = repo.FindOwnedByUser(context.Background(), anotherUser.ID, item.ID)
	if err == nil || got != nil {
		t.Fatalf("repo.FindOwnedByUser(non-owner) = %+v, %v; want nil, error", got, err)
	}
}

func TestLineItemRepo_CreateMany(t *testing.T) {
	tx := testDB.Begin()
	defer tx.Rollback()

	repo := adminstore.NewLineItemRepo(tx)

	app := dbtest.CreateApp(t, tx)
	demandSource := dbtest.CreateDemandSource(t, tx, func(source *db.DemandSource) {
		source.APIKey = string(adapter.ApplovinKey)
		source.HumanName = source.APIKey
	})
	account := dbtest.CreateDemandSourceAccount(t, tx, func(acc *db.DemandSourceAccount) {
		acc.DemandSource = demandSource
	})

	items := []admin.LineItemAttrs{
		{
			HumanName:   "item-1",
			AppID:       app.ID,
			BidFloor:    ptr(decimal.NewFromFloat(0.1)),
			AdType:      ad.BannerType,
			Format:      ptr(ad.BannerFormat),
			AccountID:   account.ID,
			AccountType: account.Type,
			Extra:       map[string]any{"ad_unit_id": "one"},
		},
		{
			HumanName:   "item-2",
			AppID:       app.ID,
			BidFloor:    ptr(decimal.NewFromFloat(0.2)),
			AdType:      ad.InterstitialType,
			AccountID:   account.ID,
			AccountType: account.Type,
			Extra:       map[string]any{"ad_unit_id": "two"},
		},
	}

	if err := repo.CreateMany(context.Background(), items); err != nil {
		t.Fatalf("repo.CreateMany() error: %v", err)
	}

	got, err := repo.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("repo.List() error: %v", err)
	}
	if got.Meta.TotalCount != 2 {
		t.Fatalf("repo.CreateMany() total count = %d, want 2", got.Meta.TotalCount)
	}
}
