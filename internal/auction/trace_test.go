package auction_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/auction"
	"github.com/bidon-io/bidon-backend/internal/auction/mocks"
	"github.com/bidon-io/bidon-backend/internal/bidding"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	biddingmocks "github.com/bidon-io/bidon-backend/internal/bidding/mocks"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event/engine"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/bidon-io/bidon-backend/internal/segment"
	segmentmocks "github.com/bidon-io/bidon-backend/internal/segment/mocks"
	"github.com/bidon-io/bidon-backend/internal/telemetry"
)

type scriptedAdapter struct {
	response *adapters.DemandResponse
}

func (a scriptedAdapter) BuildImpression(openrtb.BidRequest, *schema.AuctionRequest) (*openrtb2.Imp, adapters.RTBRequestOptions, error) {
	return &openrtb2.Imp{ID: "imp-1"}, adapters.RTBRequestOptions{}, nil
}

func (a scriptedAdapter) ExecuteRequest(context.Context, *http.Client, openrtb.BidRequest) *adapters.DemandResponse {
	return a.response
}

func (a scriptedAdapter) ParseBids(dr *adapters.DemandResponse) (*adapters.DemandResponse, error) {
	return dr, nil
}

func TestService_Run_AuctionAndDSPSpans(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(prev)
	})

	auctionConfig := &auction.Config{
		ID:         1,
		UID:        "config_uid",
		PriceFloor: 0.05,
		Timeout:    15000,
	}
	keys := []adapter.Key{adapter.BidmachineKey, adapter.MetaKey, adapter.VungleKey}
	responses := map[adapter.Key]*adapters.DemandResponse{
		adapter.BidmachineKey: {
			DemandID: adapter.BidmachineKey,
			Status:   http.StatusOK,
			Bid:      &adapters.DemandBid{Price: 2.5},
		},
		adapter.MetaKey: {
			DemandID: adapter.MetaKey,
			Status:   http.StatusNoContent,
		},
		adapter.VungleKey: {
			DemandID: adapter.VungleKey,
			Error:    context.DeadlineExceeded,
		},
	}
	wantOutcome := map[adapter.Key]telemetry.Outcome{
		adapter.BidmachineKey: telemetry.OutcomeBid,
		adapter.MetaKey:       telemetry.OutcomeNoBid,
		adapter.VungleKey:     telemetry.OutcomeTimeout,
	}

	adaptersCfg := schema.Adapters{}
	demands := map[adapter.Key]map[string]any{}
	cfgs := adapter.ProcessedConfigsMap{}
	for _, key := range keys {
		adaptersCfg[key] = schema.Adapter{Version: "1.0.0", SDKVersion: "1.0.0"}
		demands[key] = map[string]any{"token": "token"}
		cfgs[key] = map[string]any{}
	}

	telEngine := &telemetry.MemoryEngine{}
	biddingBuilder := &bidding.Builder{
		AdaptersBuilder: &biddingmocks.AdaptersBuilderMock{
			BuildFunc: func(key adapter.Key, _ adapter.ProcessedConfigsMap) (*adapters.Bidder, error) {
				return &adapters.Bidder{Adapter: scriptedAdapter{response: responses[key]}, Client: http.DefaultClient}, nil
			},
		},
		NotificationHandler: &biddingmocks.NotificationHandlerMock{
			HandleBiddingRoundFunc: func(context.Context, *schema.AdObject, bidding.AuctionResult, string, string) error {
				return nil
			},
		},
		BidCacher: &biddingmocks.BidCacherMock{
			ApplyBidCacheFunc: func(_ context.Context, _ *schema.AuctionRequest, aucRes *bidding.AuctionResult) []adapters.DemandResponse {
				return aucRes.Bids
			},
		},
		Telemetry: telemetry.New(telEngine, nil),
	}

	request := &schema.AuctionRequest{
		AdType: ad.BannerType,
		AdObject: schema.AdObject{
			AuctionID:  "auc-trace",
			AuctionKey: "1ERNSV33K4000",
			PriceFloor: 0.01,
			Demands:    demands,
		},
		Adapters: adaptersCfg,
		BaseRequest: schema.BaseRequest{
			Device: schema.Device{
				OS:   "android",
				Type: "phone",
			},
			Session: schema.Session{ID: "sess-trace"},
		},
	}

	service := &auction.Service{
		AdapterKeysFetcher: &mocks.AdapterKeysFetcherMock{
			FetchEnabledAdapterKeysFunc: func(_ context.Context, _ int64, keys []adapter.Key) ([]adapter.Key, error) {
				return keys, nil
			},
		},
		ConfigFetcher: &mocks.ConfigFetcherMock{
			FetchByUIDCachedFunc: func(_ context.Context, _ int64, _, _ string) *auction.Config {
				return auctionConfig
			},
			MatchFunc: func(_ context.Context, _ int64, _ ad.Type, _ int64, _ string) (*auction.Config, error) {
				return auctionConfig, nil
			},
		},
		AuctionBuilder: &mocks.AuctionBuilderMock{
			BuildFunc: func(ctx context.Context, params *auction.BuildParams) (*auction.Result, error) {
				result, err := biddingBuilder.HoldAuction(ctx, &bidding.BuildParams{
					App:             params.App,
					AuctionRequest:  *params.AuctionRequest,
					AdapterConfigs:  cfgs,
					BiddingAdapters: keys,
					Country:         params.Country,
				})
				return &auction.Result{
					AuctionConfiguration: auctionConfig,
					CPMAdUnits:           &[]auction.AdUnit{},
					BiddingAuctionResult: &result,
				}, err
			},
		},
		SegmentMatcher: &segment.Matcher{
			Fetcher: &segmentmocks.FetcherMock{
				FetchCachedFunc: func(_ context.Context, _ int64) ([]segment.Segment, error) {
					return nil, nil
				},
			},
		},
		EventLogger: &event.Logger{Engine: &engine.Log{}},
		Telemetry:   telemetry.New(telEngine, nil),
	}

	_, err := service.Run(context.Background(), &auction.ExecutionParams{
		Req:     request,
		App:     testApp(9),
		Country: "US",
		GeoData: geocoder.GeoData{},
		Log:     func(string) {},
		LogErr:  func(error) {},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	spans := rec.Ended()
	var parent sdktrace.ReadOnlySpan
	children := make([]sdktrace.ReadOnlySpan, 0, 3)
	for _, span := range spans {
		switch span.Name() {
		case telemetry.SpanAuctionRun:
			parent = span
		case telemetry.SpanAuctionDSP:
			children = append(children, span)
		}
	}
	if parent == nil {
		t.Fatal("missing auction.run parent span")
	}
	if len(children) != 3 {
		t.Fatalf("auction.dsp children: got %d, want 3", len(children))
	}

	if got := spanAttrString(parent, telemetry.AttrAuctionID); got != "auc-trace" {
		t.Errorf("parent auction_id = %q", got)
	}
	if got := spanAttrInt64(parent, telemetry.AttrAppID); got != 9 {
		t.Errorf("parent app_id = %d, want 9", got)
	}

	parentID := parent.SpanContext()
	seenDSP := map[string]string{}
	for _, child := range children {
		if child.SpanContext().TraceID() != parentID.TraceID() {
			t.Errorf("child %s trace_id = %s, want %s", child.Name(), child.SpanContext().TraceID(), parentID.TraceID())
		}
		if child.Parent().SpanID() != parentID.SpanID() {
			t.Errorf("child parent span id = %s, want %s", child.Parent().SpanID(), parentID.SpanID())
		}
		if got := spanAttrString(child, telemetry.AttrAuctionID); got != "auc-trace" {
			t.Errorf("child auction_id = %q", got)
		}
		if got := spanAttrInt64(child, telemetry.AttrAppID); got != 9 {
			t.Errorf("child app_id = %d, want 9", got)
		}
		dsp := spanAttrString(child, telemetry.AttrDSP)
		seenDSP[dsp] = spanAttrString(child, telemetry.AttrOutcome)
	}

	for _, key := range keys {
		dsp := string(key)
		if seenDSP[dsp] != string(wantOutcome[key]) {
			t.Errorf("%s outcome = %q, want %q", dsp, seenDSP[dsp], wantOutcome[key])
		}
	}

	parentTrace := parentID.TraceID().String()
	recs := telEngine.Records()
	if len(recs) == 0 {
		t.Fatal("expected telemetry events on the same trace")
	}
	for _, rec := range recs {
		if rec.TraceID != parentTrace {
			t.Errorf("%s trace_id = %q, want %q", rec.EventName, rec.TraceID, parentTrace)
		}
	}
}

func spanAttrString(span sdktrace.ReadOnlySpan, key string) string {
	for _, attr := range span.Attributes() {
		if string(attr.Key) == key {
			return attr.Value.AsString()
		}
	}
	return ""
}

func spanAttrInt64(span sdktrace.ReadOnlySpan, key string) int64 {
	for _, attr := range span.Attributes() {
		if string(attr.Key) == key {
			return attr.Value.AsInt64()
		}
	}
	return 0
}
