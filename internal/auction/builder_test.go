package auction_test

import (
	"context"
	"testing"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/auction"
	"github.com/google/go-cmp/cmp"
)

func TestBuilder_Build(t *testing.T) {
	config := &auction.Config{
		ID: 1,
		Rounds: []auction.RoundConfig{
			{
				ID:      "ROUND_1",
				Demands: []adapter.AdapterName{adapter.ApplovinAdapter, adapter.BidmachineAdapter},
				Timeout: 15000,
			},
			{
				ID:      "ROUND_2",
				Demands: []adapter.AdapterName{adapter.UnityAdsAdapter},
				Bidding: []adapter.AdapterName{adapter.BidmachineAdapter},
				Timeout: 15000,
			},
			{
				ID:      "ROUND_3",
				Demands: []adapter.AdapterName{adapter.ApplovinAdapter},
				Timeout: 15000,
			},
			{
				ID:      "ROUND_4",
				Demands: []adapter.AdapterName{adapter.UnityAdsAdapter, adapter.ApplovinAdapter},
				Timeout: 15000,
			},
			{
				ID:      "ROUND_5",
				Bidding: []adapter.AdapterName{adapter.BidmachineAdapter},
				Timeout: 15000,
			},
		},
	}
	lineItems := []auction.LineItem{
		{ID: "test", PriceFloor: 0.1, AdUnitID: "test_id"},
	}

	configFetcher := &auction.ConfigMatcherMock{
		MatchFunc: func(ctx context.Context, appID int64, adType ad.Type) (*auction.Config, error) {
			return config, nil
		},
	}
	lineItemsMatcher := &auction.LineItemsMatcherMock{
		MatchFunc: func(ctx context.Context, params *auction.BuildParams) ([]auction.LineItem, error) {
			return lineItems, nil
		},
	}
	builder := &auction.Builder{
		ConfigMatcher:    configFetcher,
		LineItemsMatcher: lineItemsMatcher,
	}

	testCases := []struct {
		name   string
		params *auction.BuildParams
		want   *auction.Auction
	}{
		{
			name:   "One round empty",
			params: &auction.BuildParams{Adapters: []adapter.AdapterName{adapter.UnityAdsAdapter, adapter.BidmachineAdapter}},
			want: &auction.Auction{
				ConfigID:  config.ID,
				LineItems: lineItems,
				Rounds: []auction.RoundConfig{
					{ID: "ROUND_1", Demands: []adapter.AdapterName{adapter.BidmachineAdapter}, Bidding: []adapter.AdapterName{}, Timeout: 15000},
					{ID: "ROUND_2", Demands: []adapter.AdapterName{adapter.UnityAdsAdapter}, Bidding: []adapter.AdapterName{adapter.BidmachineAdapter}, Timeout: 15000},
					{ID: "ROUND_4", Demands: []adapter.AdapterName{adapter.UnityAdsAdapter}, Bidding: []adapter.AdapterName{}, Timeout: 15000},
					{ID: "ROUND_5", Demands: []adapter.AdapterName{}, Bidding: []adapter.AdapterName{adapter.BidmachineAdapter}, Timeout: 15000},
				},
			},
		},
		{
			name:   "Single adapter available",
			params: &auction.BuildParams{Adapters: []adapter.AdapterName{adapter.ApplovinAdapter}},
			want: &auction.Auction{
				ConfigID:  config.ID,
				LineItems: lineItems,
				Rounds: []auction.RoundConfig{
					{ID: "ROUND_1", Demands: []adapter.AdapterName{adapter.ApplovinAdapter}, Bidding: []adapter.AdapterName{}, Timeout: 15000},
					{ID: "ROUND_3", Demands: []adapter.AdapterName{adapter.ApplovinAdapter}, Bidding: []adapter.AdapterName{}, Timeout: 15000},
					{ID: "ROUND_4", Demands: []adapter.AdapterName{adapter.ApplovinAdapter}, Bidding: []adapter.AdapterName{}, Timeout: 15000},
				},
			},
		},
		{
			name:   "Empty Response",
			params: &auction.BuildParams{Adapters: []adapter.AdapterName{}},
			want: &auction.Auction{
				ConfigID:  config.ID,
				LineItems: lineItems,
				Rounds:    []auction.RoundConfig{},
			},
		},
	}

	for _, tC := range testCases {
		got, _ := builder.Build(context.Background(), tC.params)

		if diff := cmp.Diff(tC.want, got); diff != "" {
			t.Errorf("builder.Build -> %+v mismatch \n(-want, +got)\n%s", tC.name, diff)
		}
	}
}
