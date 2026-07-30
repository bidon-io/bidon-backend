package adapters_test

import (
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

func TestBuildRTBRequest_Defaults(t *testing.T) {
	t.Parallel()

	auctionRequest := &schema.AuctionRequest{}
	auctionRequest.Adapters = schema.Adapters{
		adapter.MolocoKey: {SDKVersion: "1.2.3"},
	}

	imp := &openrtb2.Imp{
		Banner: &openrtb2.Banner{W: ptrInt64(320), H: ptrInt64(50)},
	}
	request := openrtb.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{
			{BidFloor: 1.25},
		},
	}

	got := adapters.BuildRTBRequest(request, auctionRequest, adapter.MolocoKey, imp, adapters.RTBRequestOptions{})

	if len(got.Imp) != 1 {
		t.Fatalf("expected 1 imp, got %d", len(got.Imp))
	}
	if got.Imp[0].ID == "" {
		t.Fatal("expected non-empty imp ID")
	}
	if got.Imp[0].DisplayManager != string(adapter.MolocoKey) {
		t.Fatalf("DisplayManager = %q, want %q", got.Imp[0].DisplayManager, adapter.MolocoKey)
	}
	if got.Imp[0].DisplayManagerVer != "1.2.3" {
		t.Fatalf("DisplayManagerVer = %q, want 1.2.3", got.Imp[0].DisplayManagerVer)
	}
	if got.Imp[0].Secure == nil || *got.Imp[0].Secure != 1 {
		t.Fatalf("Secure = %v, want 1", got.Imp[0].Secure)
	}
	if got.Imp[0].BidFloor != 1.25 {
		t.Fatalf("BidFloor = %v, want 1.25", got.Imp[0].BidFloor)
	}
	if got.Imp[0].BidFloorCur != "USD" {
		t.Fatalf("BidFloorCur = %q, want USD", got.Imp[0].BidFloorCur)
	}
	if len(got.Cur) != 1 || got.Cur[0] != "USD" {
		t.Fatalf("Cur = %v, want [USD]", got.Cur)
	}
	if got.Imp[0].Banner == nil || got.Imp[0].Banner.W == nil || *got.Imp[0].Banner.W != 320 {
		t.Fatalf("expected creative banner preserved, got %+v", got.Imp[0].Banner)
	}
}

func TestBuildRTBRequest_OptionalOverrides(t *testing.T) {
	t.Parallel()

	auctionRequest := &schema.AuctionRequest{}
	auctionRequest.Adapters = schema.Adapters{
		adapter.InmobiKey: {SDKVersion: "9.9.9"},
	}

	imp := &openrtb2.Imp{}
	request := openrtb.BidRequest{ID: "req-2"}

	got := adapters.BuildRTBRequest(request, auctionRequest, adapter.InmobiKey, imp, adapters.RTBRequestOptions{
		TagID:       "placement-1",
		AppID:       "app-1",
		PublisherID: "publisher-1",
		BuyerUID:    "token-1",
	})

	if len(got.Imp) != 1 {
		t.Fatalf("expected 1 imp, got %d", len(got.Imp))
	}
	if got.Imp[0].TagID != "placement-1" {
		t.Fatalf("TagID = %q, want placement-1", got.Imp[0].TagID)
	}
	if got.App == nil || got.App.ID != "app-1" {
		t.Fatalf("App.ID = %v, want app-1", got.App)
	}
	if got.App.Publisher == nil || got.App.Publisher.ID != "publisher-1" {
		t.Fatalf("App.Publisher.ID = %v, want publisher-1", got.App.Publisher)
	}
	if got.User == nil || got.User.BuyerUID != "token-1" {
		t.Fatalf("User.BuyerUID = %v, want token-1", got.User)
	}
}

func TestBuildRTBRequest_OptOuts(t *testing.T) {
	t.Parallel()

	auctionRequest := &schema.AuctionRequest{}
	auctionRequest.Adapters = schema.Adapters{
		adapter.VKAdsKey: {SDKVersion: "3.0.0"},
	}

	imp := &openrtb2.Imp{TagID: "keep-me"}
	request := openrtb.BidRequest{ID: "req-3"}

	got := adapters.BuildRTBRequest(request, auctionRequest, adapter.VKAdsKey, imp, adapters.RTBRequestOptions{
		OmitBidFloorCur:    true,
		OmitSecure:         true,
		OmitDisplayManager: true,
	})

	if len(got.Imp) != 1 {
		t.Fatalf("expected 1 imp, got %d", len(got.Imp))
	}
	if got.Imp[0].DisplayManager != "" || got.Imp[0].DisplayManagerVer != "" {
		t.Fatalf("expected DisplayManager omitted, got %q / %q", got.Imp[0].DisplayManager, got.Imp[0].DisplayManagerVer)
	}
	if got.Imp[0].Secure != nil {
		t.Fatalf("expected Secure omitted, got %v", *got.Imp[0].Secure)
	}
	if got.Imp[0].BidFloorCur != "" {
		t.Fatalf("expected BidFloorCur omitted, got %q", got.Imp[0].BidFloorCur)
	}
	if got.Imp[0].TagID != "keep-me" {
		t.Fatalf("TagID = %q, want keep-me", got.Imp[0].TagID)
	}
	if len(got.Cur) != 1 || got.Cur[0] != "USD" {
		t.Fatalf("Cur = %v, want [USD]", got.Cur)
	}
}

func TestBuildRTBRequest_NilImp(t *testing.T) {
	t.Parallel()

	request := openrtb.BidRequest{ID: "req-4", Cur: []string{"EUR"}}
	got := adapters.BuildRTBRequest(request, &schema.AuctionRequest{}, adapter.MetaKey, nil, adapters.RTBRequestOptions{})

	if got.ID != request.ID || len(got.Cur) != 1 || got.Cur[0] != "EUR" {
		t.Fatalf("nil imp should leave request unchanged, got %+v", got)
	}
}

func ptrInt64(v int64) *int64 {
	return &v
}
