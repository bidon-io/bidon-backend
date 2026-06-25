package admin

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/admin/resource"
	v8n "github.com/go-ozzo/ozzo-validation/v4"
)

func Test_lineItemAttrsValidator_ValidateWithContext(t *testing.T) {
	tests := []struct {
		name                string
		attrs               *LineItemAttrs
		demandSourceAccount *DemandSourceAccount
		wantErr             bool
	}{
		{
			"valid Admob",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"ad_unit_id": "ca-app-pub-3940256099942544/5224354917",
					"foo":        "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.AdmobKey),
					},
				},
			},
			false,
		},
		{
			"valid Amazon",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"slot_uuid": "26069ec0-4151-4194-a181-7a0017efdf28",
					"format":    "VIDEO",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.AmazonKey),
					},
				},
			},
			false,
		},
		{
			"valid Applovin",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"zone_id": "bd706625a42e3413",
					"foo":     "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.ApplovinKey),
					},
				},
			},
			false,
		},
		{
			"valid BigoAds",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"slot_id": "10175763-10078514",
					"foo":     "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.BigoAdsKey),
					},
				},
			},
			false,
		},
		{
			"valid Chartboost",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"ad_location": "123",
					"mediation":   "appodeal",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.ChartboostKey),
					},
				},
			},
			false,
		},
		{
			"valid DT Exchange",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"spot_id": "1187213",
					"foo":     "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.DTExchangeKey),
					},
				},
			},
			false,
		},
		{
			"valid GAM",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"ad_unit_id": "/111/Bidon/Interstitials/0.4 USD",
					"foo":        "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.GAMKey),
					},
				},
			},
			false,
		},
		{
			"valid IronSource",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"instance_id": "123",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.IronSourceKey),
					},
				},
			},
			false,
		},
		{
			"valid InMobi",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"placement_id": "1621323861540",
					"foo":          "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.InmobiKey),
					},
				},
			},
			false,
		},
		{
			"valid Meta",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"placement_id": "767803077426274_1212622446277666",
					"foo":          "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.MetaKey),
					},
				},
			},
			false,
		},
		{
			"valid Unity",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"placement_id": "bidon_rv_43",
					"foo":          "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.UnityAdsKey),
					},
				},
			},
			false,
		},
		{
			"valid Zmaticoo",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"placement_id": "placement-123",
					"foo":          "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.ZmaticooKey),
					},
				},
			},
			false,
		},
		{
			"valid VK Ads",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"slot_id":   "8066185",
					"mediation": "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.VKAdsKey),
					},
				},
			},
			false,
		},
		{
			"valid Vungle",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"placement_id": "BANNER_TEST-8066185",
					"foo":          "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.VungleKey),
					},
				},
			},
			false,
		},
		{
			"valid MobileFuse",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"placement_id": "938186",
					"foo":          "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.MobileFuseKey),
					},
				},
			},
			false,
		},
		{
			"valid Mintegral",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"placement_id": "938186",
					"unit_id":      "2567735",
					"foo":          "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.MintegralKey),
					},
				},
			},
			false,
		},
		{
			"valid Yandex",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"ad_unit_id": "938186",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.YandexKey),
					},
				},
			},
			false,
		},
		{
			"valid nil Extra",
			&LineItemAttrs{
				AccountID: 1,
				Extra:     nil,
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.MintegralKey),
					},
				},
			},
			false,
		},
		{
			"valid adapter that has no required keys",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"foo": "bar",
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.BidmachineKey),
					},
				},
			},
			false,
		},
		{
			"invalid when no keys present",
			&LineItemAttrs{
				AccountID: 1,
				Extra:     map[string]any{},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.MintegralKey),
					},
				},
			},
			true,
		},
		{
			"invalid when values are not string",
			&LineItemAttrs{
				AccountID: 1,
				Extra: map[string]any{
					"placement_id": 938186,
					"ad_unit_id":   2567735,
				},
			},
			&DemandSourceAccount{
				DemandSource: DemandSource{
					DemandSourceAttrs: DemandSourceAttrs{
						ApiKey: string(adapter.MintegralKey),
					},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &DemandSourceAccountRepoMock{
				FindFunc: func(ctx context.Context, id int64) (*DemandSourceAccount, error) {
					if id != tt.attrs.AccountID {
						t.Errorf("Find() got = %v, want %v", id, tt.attrs.AccountID)
					}
					return tt.demandSourceAccount, nil
				},
			}
			v := &lineItemAttrsValidator{
				attrs:                   tt.attrs,
				demandSourceAccountRepo: repo,
			}
			if err := v.ValidateWithContext(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("ValidateWithContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLineItemService_ImportCSV_SupportedAdapters(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name   string
		apiKey adapter.Key
		csv    string
	}

	tests := []testCase{
		{
			name:   "admob",
			apiKey: adapter.AdmobKey,
			csv:    "ad_format,bid_floor,ad_unit_id\nbanner,0.4,ca-app-pub-3940256099942544/5224354917\n",
		},
		{
			name:   "applovin",
			apiKey: adapter.ApplovinKey,
			csv:    "ad_format,bid_floor,zone_id\ninterstitial,0.4,bd706625a42e3413\n",
		},
		{
			name:   "bigoads",
			apiKey: adapter.BigoAdsKey,
			csv:    "ad_format,bid_floor,slot_id\nrewarded,0.4,10175763-10078514\n",
		},
		{
			name:   "chartboost",
			apiKey: adapter.ChartboostKey,
			csv:    "ad_format,bid_floor,ad_location,mediation\nbanner,0.4,123,max\n",
		},
		{
			name:   "dt_exchange",
			apiKey: adapter.DTExchangeKey,
			csv:    "ad_format,bid_floor,spot_id\ninterstitial,0.4,1187213\n",
		},
		{
			name:   "gam",
			apiKey: adapter.GAMKey,
			csv:    "ad_format,bid_floor,ad_unit_id\nbanner,0.4,/111/Bidon/Interstitials/0.4 USD\n",
		},
		{
			name:   "meta",
			apiKey: adapter.MetaKey,
			csv:    "ad_format,bid_floor,placement_id\nrewarded,0.4,767803077426274_1212622446277666\n",
		},
		{
			name:   "mintegral",
			apiKey: adapter.MintegralKey,
			csv:    "ad_format,bid_floor,placement_id,unit_id\nrewarded,0.4,938186,2567735\n",
		},
		{
			name:   "inmobi",
			apiKey: adapter.InmobiKey,
			csv:    "ad_format,bid_floor,placement_id\ninterstitial,0.4,1621323861540\n",
		},
		{
			name:   "unityads",
			apiKey: adapter.UnityAdsKey,
			csv:    "ad_format,bid_floor,placement_id\nrewarded,0.4,bidon_rv_43\n",
		},
		{
			name:   "vkads",
			apiKey: adapter.VKAdsKey,
			csv:    "ad_format,bid_floor,slot_id,mediation\nbanner,0.4,8066185,max\n",
		},
		{
			name:   "vungle",
			apiKey: adapter.VungleKey,
			csv:    "ad_format,bid_floor,placement_id\nbanner,0.4,BANNER_TEST-8066185\n",
		},
		{
			name:   "yandex",
			apiKey: adapter.YandexKey,
			csv:    "ad_format,bid_floor,ad_unit_id\nbanner,0.4,938186\n",
		},
		{
			name:   "zmaticoo",
			apiKey: adapter.ZmaticooKey,
			csv:    "ad_format,bid_floor,placement_id\ninterstitial,0.4,placement-123\n",
		},
		{
			name:   "ironsource",
			apiKey: adapter.IronSourceKey,
			csv:    "ad_format,bid_floor,instance_id\nrewarded,0.4,instance-123\n",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var created []LineItemAttrs
			service := &LineItemService{
				store: &StoreMock{
					DemandSourceAccountsFunc: func() DemandSourceAccountRepo {
						return &DemandSourceAccountRepoMock{
							FindFunc: func(_ context.Context, _ int64) (*DemandSourceAccount, error) {
								return &DemandSourceAccount{
									ID: 10,
									DemandSourceAccountAttrs: DemandSourceAccountAttrs{
										Type: "DemandSourceAccount::test",
									},
									DemandSource: DemandSource{
										DemandSourceAttrs: DemandSourceAttrs{
											ApiKey: string(tt.apiKey),
										},
									},
								}, nil
							},
						}
					},
					LineItemsFunc: func() LineItemRepo {
						return &lineItemRepoStub{
							createManyFn: func(_ context.Context, items []LineItemAttrs) error {
								created = items
								return nil
							},
						}
					},
				},
			}

			err := service.ImportCSV(context.Background(), nil, strings.NewReader(tt.csv), LineItemImportCSVAttrs{
				AppID:     100,
				AccountID: 10,
				IsBidding: true,
			})
			if err != nil {
				t.Fatalf("ImportCSV() error = %v", err)
			}
			if len(created) != 1 {
				t.Fatalf("ImportCSV() created %d items, want 1", len(created))
			}
			if created[0].AppID != 100 {
				t.Fatalf("ImportCSV() AppID = %d, want 100", created[0].AppID)
			}
			if created[0].AccountID != 10 {
				t.Fatalf("ImportCSV() AccountID = %d, want 10", created[0].AccountID)
			}
			if created[0].IsBidding == nil || !*created[0].IsBidding {
				t.Fatalf("ImportCSV() IsBidding = %v, want true", created[0].IsBidding)
			}
		})
	}
}

func TestLineItemService_ImportCSV_ErrorCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		apiKey  string
		csv     string
		wantErr string
	}{
		{
			name:    "unsupported demand source",
			apiKey:  "unknown",
			csv:     "ad_format,bid_floor,ad_unit_id\nbanner,0.4,id\n",
			wantErr: "unsupported demand source",
		},
		{
			name:    "empty csv",
			apiKey:  string(adapter.AdmobKey),
			csv:     "ad_format,bid_floor,ad_unit_id\n",
			wantErr: "csv empty",
		},
		{
			name:    "unknown ad format",
			apiKey:  string(adapter.AdmobKey),
			csv:     "ad_format,bid_floor,ad_unit_id\nnative,0.4,id\n",
			wantErr: "build line item attrs",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &LineItemService{
				store: &StoreMock{
					DemandSourceAccountsFunc: func() DemandSourceAccountRepo {
						return &DemandSourceAccountRepoMock{
							FindFunc: func(_ context.Context, _ int64) (*DemandSourceAccount, error) {
								return &DemandSourceAccount{
									ID: 10,
									DemandSourceAccountAttrs: DemandSourceAccountAttrs{
										Type: "DemandSourceAccount::test",
									},
									DemandSource: DemandSource{
										DemandSourceAttrs: DemandSourceAttrs{
											ApiKey: tt.apiKey,
										},
									},
								}, nil
							},
						}
					},
					LineItemsFunc: func() LineItemRepo {
						return &lineItemRepoStub{}
					},
				},
			}

			err := service.ImportCSV(context.Background(), nil, strings.NewReader(tt.csv), LineItemImportCSVAttrs{
				AppID:     100,
				AccountID: 10,
				IsBidding: true,
			})
			if err == nil {
				t.Fatalf("ImportCSV() error = nil, want contains %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ImportCSV() error = %q, want contains %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestLineItemService_NewServiceAndPolicyMethods(t *testing.T) {
	t.Parallel()

	store := &StoreMock{
		AppsFunc: func() AppRepo {
			return &AppRepoMock{
				FindOwnedByUserFunc: func(_ context.Context, _ int64, id int64) (*App, error) {
					return &App{ID: id}, nil
				},
			}
		},
		UsersFunc: func() UserRepo {
			return &UserRepoMock{}
		},
		DemandSourcesFunc: func() DemandSourceRepo {
			return &DemandSourceRepoMock{}
		},
		DemandSourceAccountsFunc: func() DemandSourceAccountRepo {
			return &DemandSourceAccountRepoMock{
				FindOwnedByUserOrSharedFunc: func(_ context.Context, _ int64, _ int64) (*DemandSourceAccount, error) {
					return &DemandSourceAccount{}, nil
				},
			}
		},
		LineItemsFunc: func() LineItemRepo {
			return &lineItemRepoStub{}
		},
	}

	service := NewLineItemService(store)
	if service == nil {
		t.Fatalf("NewLineItemService() = nil, want non-nil service")
	}

	policy := newLineItemPolicy(store)
	authCtx := lineItemTestAuthCtx{userID: 42, isAdmin: false}
	_ = policy.getReadScope(authCtx)
	_ = policy.getManageScope(authCtx)

	if err := policy.authorizeCreate(context.Background(), authCtx, &LineItemAttrs{
		AppID:     11,
		AccountID: 22,
	}); err != nil {
		t.Fatalf("authorizeCreate() error = %v", err)
	}

	profile := &LineItem{
		ID: 1,
		LineItemAttrs: LineItemAttrs{
			AppID:     11,
			AccountID: 22,
		},
	}
	if err := policy.authorizeUpdate(context.Background(), authCtx, profile, &LineItemAttrs{
		AppID:     12,
		AccountID: 23,
	}); err != nil {
		t.Fatalf("authorizeUpdate() error = %v", err)
	}

	if err := policy.authorizeDelete(context.Background(), authCtx, profile); err != nil {
		t.Fatalf("authorizeDelete() error = %v", err)
	}

	perms := policy.permissions(authCtx)
	if !perms.Read || !perms.Create {
		t.Fatalf("permissions() = %+v, want read/create true", perms)
	}

	instancePerms := policy.instancePermissions(authCtx, profile)
	if !instancePerms.Update || !instancePerms.Delete {
		t.Fatalf("instancePermissions() = %+v, want update/delete true", instancePerms)
	}
}

func TestLineItemService_Update_DuplicateByKey(t *testing.T) {
	t.Parallel()

	updateCalled := false
	lineItemRepo := &lineItemRepoStub{
		findOwnedByUserFn: func(_ context.Context, _ int64, id int64) (*LineItem, error) {
			return &LineItem{
				ID: id,
				LineItemAttrs: LineItemAttrs{
					AppID:       11,
					AccountType: "DemandSourceAccount::admob",
					AccountID:   22,
					AdType:      ad.BannerType,
					Format:      ptr(ad.BannerFormat),
					Extra:       map[string]any{"ad_unit_id": "old"},
				},
			}, nil
		},
		findDuplicateFn: func(_ context.Context, _ *LineItemAttrs, _ int64) (int64, error) {
			return 999, nil
		},
		updateFn: func(_ context.Context, _ int64, _ *LineItemAttrs) (*LineItem, error) {
			updateCalled = true
			return &LineItem{}, nil
		},
	}

	store := &StoreMock{
		LineItemsFunc: func() LineItemRepo { return lineItemRepo },
		DemandSourceAccountsFunc: func() DemandSourceAccountRepo {
			return &DemandSourceAccountRepoMock{
				FindFunc: func(_ context.Context, _ int64) (*DemandSourceAccount, error) {
					return &DemandSourceAccount{
						DemandSource: DemandSource{
							DemandSourceAttrs: DemandSourceAttrs{ApiKey: string(adapter.AdmobKey)},
						},
					}, nil
				},
			}
		},
		AppsFunc:          func() AppRepo { return &AppRepoMock{} },
		UsersFunc:         func() UserRepo { return &UserRepoMock{} },
		DemandSourcesFunc: func() DemandSourceRepo { return &DemandSourceRepoMock{} },
	}
	service := NewLineItemService(store)

	_, err := service.Update(context.Background(), lineItemTestAuthCtx{userID: 1, isAdmin: false}, 123, &LineItemAttrs{
		AppID:       11,
		AccountType: "DemandSourceAccount::admob",
		AccountID:   22,
		AdType:      ad.BannerType,
		Format:      ptr(ad.BannerFormat),
		Extra:       map[string]any{"ad_unit_id": "new"},
	})
	if err == nil {
		t.Fatalf("Update() error = nil, want duplicate validation error")
	}

	var validationErr v8n.Errors
	if !errors.As(err, &validationErr) {
		t.Fatalf("Update() error type = %T, want v8n.Errors", err)
	}
	if updateCalled {
		t.Fatalf("Update() called repo.Update when duplicate exists")
	}
}

func TestLineItemService_Update_NoDuplicate(t *testing.T) {
	t.Parallel()

	updateCalled := false
	lineItemRepo := &lineItemRepoStub{
		findOwnedByUserFn: func(_ context.Context, _ int64, id int64) (*LineItem, error) {
			return &LineItem{
				ID: id,
				LineItemAttrs: LineItemAttrs{
					AppID:       11,
					AccountType: "DemandSourceAccount::admob",
					AccountID:   22,
					AdType:      ad.BannerType,
					Format:      ptr(ad.BannerFormat),
					Extra:       map[string]any{"ad_unit_id": "old"},
				},
			}, nil
		},
		findDuplicateFn: func(_ context.Context, _ *LineItemAttrs, _ int64) (int64, error) {
			return 0, nil
		},
		updateFn: func(_ context.Context, id int64, attrs *LineItemAttrs) (*LineItem, error) {
			updateCalled = true
			return &LineItem{ID: id, LineItemAttrs: *attrs}, nil
		},
	}

	store := &StoreMock{
		LineItemsFunc: func() LineItemRepo { return lineItemRepo },
		DemandSourceAccountsFunc: func() DemandSourceAccountRepo {
			return &DemandSourceAccountRepoMock{
				FindFunc: func(_ context.Context, _ int64) (*DemandSourceAccount, error) {
					return &DemandSourceAccount{
						DemandSource: DemandSource{
							DemandSourceAttrs: DemandSourceAttrs{ApiKey: string(adapter.AdmobKey)},
						},
					}, nil
				},
			}
		},
		AppsFunc:          func() AppRepo { return &AppRepoMock{} },
		UsersFunc:         func() UserRepo { return &UserRepoMock{} },
		DemandSourcesFunc: func() DemandSourceRepo { return &DemandSourceRepoMock{} },
	}
	service := NewLineItemService(store)

	got, err := service.Update(context.Background(), lineItemTestAuthCtx{userID: 1, isAdmin: false}, 123, &LineItemAttrs{
		AppID:       11,
		AccountType: "DemandSourceAccount::admob",
		AccountID:   22,
		AdType:      ad.BannerType,
		Format:      ptr(ad.BannerFormat),
		Extra:       map[string]any{"ad_unit_id": "new"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !updateCalled {
		t.Fatalf("Update() did not call repo.Update")
	}
	if got == nil || got.ID != 123 {
		t.Fatalf("Update() = %+v, want id 123", got)
	}
	if !got.Permissions.Update || !got.Permissions.Delete {
		t.Fatalf("Update() permissions = %+v, want update/delete true", got.Permissions)
	}
}

func TestLineItemService_Update_repoError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("update failed")
	lineItemRepo := &lineItemRepoStub{
		findOwnedByUserFn: func(_ context.Context, _ int64, id int64) (*LineItem, error) {
			return &LineItem{
				ID: id,
				LineItemAttrs: LineItemAttrs{
					AppID:       11,
					AccountType: "DemandSourceAccount::admob",
					AccountID:   22,
					AdType:      ad.BannerType,
					Format:      ptr(ad.BannerFormat),
					Extra:       map[string]any{"ad_unit_id": "old"},
				},
			}, nil
		},
		findDuplicateFn: func(_ context.Context, _ *LineItemAttrs, _ int64) (int64, error) {
			return 0, nil
		},
		updateFn: func(_ context.Context, _ int64, _ *LineItemAttrs) (*LineItem, error) {
			return nil, repoErr
		},
	}

	store := &StoreMock{
		LineItemsFunc: func() LineItemRepo { return lineItemRepo },
		DemandSourceAccountsFunc: func() DemandSourceAccountRepo {
			return &DemandSourceAccountRepoMock{
				FindFunc: func(_ context.Context, _ int64) (*DemandSourceAccount, error) {
					return &DemandSourceAccount{
						DemandSource: DemandSource{
							DemandSourceAttrs: DemandSourceAttrs{ApiKey: string(adapter.AdmobKey)},
						},
					}, nil
				},
			}
		},
		AppsFunc:          func() AppRepo { return &AppRepoMock{} },
		UsersFunc:         func() UserRepo { return &UserRepoMock{} },
		DemandSourcesFunc: func() DemandSourceRepo { return &DemandSourceRepoMock{} },
	}
	service := NewLineItemService(store)

	got, err := service.Update(context.Background(), lineItemTestAuthCtx{userID: 1, isAdmin: false}, 123, &LineItemAttrs{
		Extra: map[string]any{"ad_unit_id": "new"},
	})
	if !errors.Is(err, repoErr) {
		t.Fatalf("Update() error = %v, want %v", err, repoErr)
	}
	if got != nil {
		t.Fatalf("Update() = %+v, want nil", got)
	}
}

func TestLineItemService_Update_UsesMergedAttrsForDuplicateValidation(t *testing.T) {
	t.Parallel()

	lineItemRepo := &lineItemRepoStub{
		findOwnedByUserFn: func(_ context.Context, _ int64, id int64) (*LineItem, error) {
			return &LineItem{
				ID: id,
				LineItemAttrs: LineItemAttrs{
					AppID:       11,
					AccountType: "DemandSourceAccount::admob",
					AccountID:   22,
					AdType:      ad.BannerType,
					Format:      ptr(ad.BannerFormat),
					Extra:       map[string]any{"ad_unit_id": "old"},
				},
			}, nil
		},
		findDuplicateFn: func(_ context.Context, attrs *LineItemAttrs, excludeID int64) (int64, error) {
			if excludeID != 123 {
				t.Fatalf("excludeID = %d, want 123", excludeID)
			}
			if attrs.AppID != 11 || attrs.AccountID != 22 || attrs.AdType != ad.BannerType {
				t.Fatalf("merged attrs not used, got %+v", attrs)
			}
			if attrs.Extra["ad_unit_id"] != "new" {
				t.Fatalf("merged attrs extra not overridden, got %+v", attrs.Extra)
			}
			return 999, nil
		},
		updateFn: func(_ context.Context, _ int64, _ *LineItemAttrs) (*LineItem, error) {
			t.Fatalf("repo.Update should not be called on duplicate")
			return nil, nil
		},
	}

	store := &StoreMock{
		LineItemsFunc: func() LineItemRepo { return lineItemRepo },
		DemandSourceAccountsFunc: func() DemandSourceAccountRepo {
			return &DemandSourceAccountRepoMock{
				FindFunc: func(_ context.Context, _ int64) (*DemandSourceAccount, error) {
					return &DemandSourceAccount{
						DemandSource: DemandSource{
							DemandSourceAttrs: DemandSourceAttrs{ApiKey: string(adapter.AdmobKey)},
						},
					}, nil
				},
			}
		},
		AppsFunc:          func() AppRepo { return &AppRepoMock{} },
		UsersFunc:         func() UserRepo { return &UserRepoMock{} },
		DemandSourcesFunc: func() DemandSourceRepo { return &DemandSourceRepoMock{} },
	}
	service := NewLineItemService(store)

	_, err := service.Update(context.Background(), lineItemTestAuthCtx{userID: 1, isAdmin: false}, 123, &LineItemAttrs{
		Extra: map[string]any{"ad_unit_id": "new"},
	})
	if err == nil {
		t.Fatalf("Update() error = nil, want duplicate validation error")
	}
}

type lineItemTestAuthCtx struct {
	userID  int64
	isAdmin bool
}

func (a lineItemTestAuthCtx) UserID() int64 {
	return a.userID
}

func (a lineItemTestAuthCtx) IsAdmin() bool {
	return a.isAdmin
}

type lineItemRepoStub struct {
	createManyFn      func(ctx context.Context, items []LineItemAttrs) error
	findOwnedByUserFn func(ctx context.Context, userID int64, id int64) (*LineItem, error)
	updateFn          func(ctx context.Context, id int64, attrs *LineItemAttrs) (*LineItem, error)
	findDuplicateFn   func(ctx context.Context, attrs *LineItemAttrs, excludeID int64) (int64, error)
}

func (s *lineItemRepoStub) List(_ context.Context, _ map[string][]string) (*resource.Collection[LineItem], error) {
	return nil, nil
}

func (s *lineItemRepoStub) ListOwnedByUser(_ context.Context, _ int64, _ map[string][]string) (*resource.Collection[LineItem], error) {
	return nil, nil
}

func (s *lineItemRepoStub) Find(_ context.Context, _ int64) (*LineItem, error) {
	return nil, nil
}

func (s *lineItemRepoStub) FindOwnedByUser(ctx context.Context, userID int64, id int64) (*LineItem, error) {
	if s.findOwnedByUserFn != nil {
		return s.findOwnedByUserFn(ctx, userID, id)
	}
	return nil, nil
}

func (s *lineItemRepoStub) Create(_ context.Context, _ *LineItemAttrs) (*LineItem, error) {
	return nil, nil
}

func (s *lineItemRepoStub) Update(ctx context.Context, id int64, attrs *LineItemAttrs) (*LineItem, error) {
	if s.updateFn != nil {
		return s.updateFn(ctx, id, attrs)
	}
	return nil, nil
}

func (s *lineItemRepoStub) Delete(_ context.Context, _ int64) error {
	return nil
}

func (s *lineItemRepoStub) CreateMany(ctx context.Context, items []LineItemAttrs) error {
	if s.createManyFn != nil {
		return s.createManyFn(ctx, items)
	}
	return nil
}

func (s *lineItemRepoStub) FindDuplicateIDByAttrs(ctx context.Context, attrs *LineItemAttrs, excludeID int64) (int64, error) {
	if s.findDuplicateFn != nil {
		return s.findDuplicateFn(ctx, attrs, excludeID)
	}
	return 0, nil
}
