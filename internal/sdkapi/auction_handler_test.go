package sdkapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/auction"
	auctionmocks "github.com/bidon-io/bidon-backend/internal/auction/mocks"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
	sdkapimocks "github.com/bidon-io/bidon-backend/internal/sdkapi/mocks"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/bidon-io/bidon-backend/internal/segment"
	segmentmocks "github.com/bidon-io/bidon-backend/internal/segment/mocks"
)

func TestAuctionHandler_Handle(t *testing.T) {
	// Create a new Echo instance
	e := echo.New()
	app := App{ID: 1}
	appFetcher := &sdkapimocks.AppFetcherMock{
		FetchFunc: func(ctx context.Context, appKey string, appBundle string) (App, error) {
			return app, nil
		},
	}
	gcoder := &sdkapimocks.GeocoderMock{
		LookupFunc: func(ctx context.Context, ipString string) (geocoder.GeoData, error) {
			return geocoder.GeoData{}, nil
		},
	}
	config := &auction.Config{
		ID: 1,
		Rounds: []auction.RoundConfig{
			{
				ID:      "ROUND_1",
				Demands: []adapter.Key{adapter.ApplovinKey, adapter.BidmachineKey},
				Timeout: 15000,
			},
			{
				ID:      "ROUND_2",
				Demands: []adapter.Key{adapter.UnityAdsKey},
				Bidding: []adapter.Key{adapter.BidmachineKey},
				Timeout: 15000,
			},
			{
				ID:      "ROUND_3",
				Demands: []adapter.Key{adapter.ApplovinKey},
				Timeout: 15000,
			},
			{
				ID:      "ROUND_4",
				Demands: []adapter.Key{adapter.UnityAdsKey, adapter.ApplovinKey},
				Timeout: 15000,
			},
			{
				ID:      "ROUND_5",
				Bidding: []adapter.Key{adapter.BidmachineKey},
				Timeout: 15000,
			},
		},
	}
	lineItems := []auction.LineItem{
		{ID: "test", PriceFloor: 0.1, AdUnitID: "test_id"},
	}
	segments := []segment.Segment{
		{ID: 1, Filters: []segment.Filter{{Type: "country", Name: "country", Operator: "in", Values: []string{"US", "UK"}}}},
	}

	configMatcher := &auctionmocks.ConfigMatcherMock{
		MatchFunc: func(ctx context.Context, appID int64, adType ad.Type, segmentID int64) (*auction.Config, error) {
			return config, nil
		},
	}
	lineItemsMatcher := &auctionmocks.LineItemsMatcherMock{
		MatchFunc: func(ctx context.Context, params *auction.BuildParams) ([]auction.LineItem, error) {
			return lineItems, nil
		},
	}
	segmentFetcher := &segmentmocks.FetcherMock{
		FetchFunc: func(ctx context.Context, appID int64) ([]segment.Segment, error) {
			return segments, nil
		},
	}
	auctionBuilder := &auction.Builder{
		ConfigMatcher:    configMatcher,
		LineItemsMatcher: lineItemsMatcher,
	}
	segmentMatcher := &segment.Matcher{
		Fetcher: segmentFetcher,
	}

	// Create a new AuctionHandler instance
	handler := &AuctionHandler{
		BaseHandler: &BaseHandler[schema.AuctionRequest, *schema.AuctionRequest]{
			AppFetcher: appFetcher,
			Geocoder:   gcoder,
		},
		AuctionBuilder: auctionBuilder,
		SegmentMatcher: segmentMatcher,
	}

	// Create a new HTTP request
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	// Create a new HTTP response recorder
	rec := httptest.NewRecorder()

	// Create a new Echo context
	c := e.NewContext(req, rec)

	// Call the Handle method
	err := handler.Handle(c)
	if err != nil {
		t.Fatalf("Handle method returned an error: %v", err)
	}

	// Assert that the response status code is HTTP 200 OK
	assert.Equal(t, http.StatusOK, rec.Code)

	// Assert any other response expectations based on the implementation
	// and the specific scenario you want to test
}
