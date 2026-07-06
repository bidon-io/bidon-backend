package dspsimulator_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/dspsimulator"
)

var testdataDir = "../sdkapi/v2/apihandlers/testdata/auction/adikteev"

func TestResponseStore_LoadAndLookup(t *testing.T) {
	store, err := dspsimulator.NewResponseStore(testdataDir)
	if err != nil {
		t.Fatalf("NewResponseStore: %v", err)
	}

	expectedKeys := []string{
		"adikteev/android/banner/320x50",
		"adikteev/android/banner/300x250",
		"adikteev/android/banner/320x480",
		"adikteev/android/banner/320x480/mraid",
		"adikteev/android/video/vast",
		"adikteev/android/native",
		"adikteev/ios/banner/320x50",
		"adikteev/ios/banner/300x250",
		"adikteev/ios/banner/320x480",
		"adikteev/ios/banner/320x480/mraid",
		"adikteev/ios/video/vast",
		"adikteev/ios/native",
	}

	for _, key := range expectedKeys {
		resp := store.Lookup(key)
		if resp == nil {
			t.Errorf("key %q not found", key)
		}
	}
}

func TestHandleBidRequest_BannerBanner(t *testing.T) {
	store, err := dspsimulator.NewResponseStore(testdataDir)
	if err != nil {
		t.Fatalf("NewResponseStore: %v", err)
	}
	svc := dspsimulator.NewService(store)

	req := loadBidRequest(t, "android_adikteev_banner_bidreq.json")
	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if resp.ID != req.ID {
		t.Errorf("response ID = %q, want %q", resp.ID, req.ID)
	}
	if len(resp.SeatBid) == 0 || len(resp.SeatBid[0].Bid) == 0 {
		t.Fatal("expected at least one seatbid with a bid")
	}
	bid := resp.SeatBid[0].Bid[0]
	if bid.ImpID != req.Imp[0].ID {
		t.Errorf("bid.ImpID = %q, want %q", bid.ImpID, req.Imp[0].ID)
	}
}

func TestHandleBidRequest_BannerMrec(t *testing.T) {
	store, err := dspsimulator.NewResponseStore(testdataDir)
	if err != nil {
		t.Fatalf("NewResponseStore: %v", err)
	}
	svc := dspsimulator.NewService(store)

	req := loadBidRequest(t, "android_adikteev_banner_mrec_bidreq.json")
	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.ID != req.ID {
		t.Errorf("response ID = %q, want %q", resp.ID, req.ID)
	}
	if len(resp.SeatBid) == 0 || len(resp.SeatBid[0].Bid) == 0 {
		t.Fatal("expected at least one seatbid with a bid")
	}
	bid := resp.SeatBid[0].Bid[0]
	if bid.ImpID != req.Imp[0].ID {
		t.Errorf("bid.ImpID = %q, want %q", bid.ImpID, req.Imp[0].ID)
	}
}

func TestHandleBidRequest_Interstitial(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := loadBidRequest(t, "android_adikteev_interstitial_bidreq.json")
	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestHandleBidRequest_InterstitialMraid(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := loadBidRequest(t, "android_adikteev_interstitial_bidreq.json")
	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestHandleBidRequest_Rewarded(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := loadBidRequest(t, "android_adikteev_rewarded_bidreq.json")
	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestHandleBidRequest_NoBid(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := &openrtb2.BidRequest{
		ID: "empty-test",
		Imp: []openrtb2.Imp{
			{
				DisplayManager: "unknown_dsp",
			},
		},
	}

	if resp := svc.HandleBidRequest(req); resp != nil {
		t.Error("expected nil response for unknown DSP")
	}
}

func TestBuildKeys_Variants(t *testing.T) {
	tests := []struct {
		name string
		imp  openrtb2.Imp
		os   string
		want []string
	}{
		{
			name: "banner with MRAID api",
			imp: openrtb2.Imp{
				DisplayManager: "adikteev",
				Banner: &openrtb2.Banner{
					W:   ptrInt64(320),
					H:   ptrInt64(480),
					API: []adcom1.APIFramework{3, 5},
				},
			},
			os:   "android",
			want: []string{"adikteev/android/banner/320x480", "adikteev/android/banner/320x480/mraid"},
		},
		{
			name: "video with VAST mimes",
			imp: openrtb2.Imp{
				DisplayManager: "adikteev",
				Video: &openrtb2.Video{
					W:     360,
					H:     640,
					MIMEs: []string{"video/mp4", "video/x-m4v", "video/quicktime", "video/mpeg", "video/avi"},
				},
			},
			os:   "android",
			want: []string{"adikteev/android/video/vast"},
		},
		{
			name: "native imp",
			imp: openrtb2.Imp{
				DisplayManager: "adikteev",
				Native:         &openrtb2.Native{},
			},
			os:   "ios",
			want: []string{"adikteev/ios/native"},
		},
		{
			name: "multi-format: video + native",
			imp: openrtb2.Imp{
				DisplayManager: "adikteev",
				Video: &openrtb2.Video{
					W:     320,
					H:     480,
					MIMEs: []string{"video/mp4"},
				},
				Native: &openrtb2.Native{},
			},
			os:   "ios",
			want: []string{"adikteev/ios/video/vast", "adikteev/ios/native"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &openrtb2.BidRequest{
				ID: "test",
				Device: &openrtb2.Device{
					OS: tt.os,
				},
				Imp: []openrtb2.Imp{tt.imp},
			}
			got := dspsimulator.BuildKeys(req)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("BuildKeys mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func loadBidRequest(t *testing.T, filename string) *openrtb2.BidRequest {
	t.Helper()
	path := filepath.Join(testdataDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	var req openrtb2.BidRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	return &req
}

func ptrInt64(v int64) *int64 {
	return &v
}

func mustReadJSON(t *testing.T, filename string) []byte {
	t.Helper()
	path := filepath.Join(testdataDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	return data
}

func TestIDRewriting(t *testing.T) {
	store, err := dspsimulator.NewResponseStore(testdataDir)
	if err != nil {
		t.Fatalf("NewResponseStore: %v", err)
	}
	svc := dspsimulator.NewService(store)

	newRequestID := "live-request-id-xxx"
	newImpID := "live-imp-id-yyy"

	req := &openrtb2.BidRequest{
		ID: newRequestID,
		Device: &openrtb2.Device{
			OS: "android",
		},
		Imp: []openrtb2.Imp{
			{
				ID:             newImpID,
				DisplayManager: "adikteev",
				Banner: &openrtb2.Banner{
					W: ptrInt64(320),
					H: ptrInt64(50),
				},
			},
		},
	}

	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if resp.ID != newRequestID {
		t.Errorf("Response.ID = %q, want %q", resp.ID, newRequestID)
	}

	if len(resp.SeatBid) == 0 || len(resp.SeatBid[0].Bid) == 0 {
		t.Fatal("expected at least one seatbid with a bid")
	}
	bid := resp.SeatBid[0].Bid[0]
	if bid.ImpID != newImpID {
		t.Errorf("Bid.ImpID = %q, want %q (was originally %s from fixture)",
			bid.ImpID, newImpID, "50a642f6-d17c-4803-bbe5-090729ed2ecc")
	}

	if bid.Price == 0 {
		t.Error("bid price should be non-zero")
	}
}

func TestResponseMatchesOneOfCandidates(t *testing.T) {
	store, err := dspsimulator.NewResponseStore(testdataDir)
	if err != nil {
		t.Fatalf("NewResponseStore: %v", err)
	}
	svc := dspsimulator.NewService(store)

	req := loadBidRequest(t, "android_adikteev_interstitial_bidreq.json")
	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	expectedPrices := []float64{11.817}

	bid := resp.SeatBid[0].Bid[0]
	found := false
	for _, p := range expectedPrices {
		if bid.Price == p {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("price %f not in expected set %v", bid.Price, expectedPrices)
	}

	if resp.ID != req.ID {
		t.Errorf("Response.ID = %q, want %q", resp.ID, req.ID)
	}
}

func TestHandleBidRequest_IosRewardedMultiFormat(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := loadBidRequest(t, "ios_adikteev_rewarded_bidreq.json")
	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response for iOS rewarded (multi-format imp)")
	}
}

func TestHandleBidRequest_RewardedNative(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := loadBidRequest(t, "android_adikteev_rewarded_bidreq.json")
	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestHandleBidRequest_IosBanner(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := loadBidRequest(t, "ios_adikteev_banner_bidreq.json")
	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response for iOS banner")
	}
}

func TestHandleBidRequest_IosBannerMrec(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := loadBidRequest(t, "ios_adikteev_banner_mrec_bidreq.json")
	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response for iOS banner mrec")
	}
}

func TestHandleBidRequest_IosInterstitial(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := loadBidRequest(t, "ios_adikteev_interstitial_bidreq.json")
	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response for iOS interstitial")
	}
}

func TestHandleBidRequest_IosInterstitialMraid(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := loadBidRequest(t, "ios_adikteev_interstitial_bidreq.json")
	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response for iOS interstitial mraid")
	}
}

func TestHandleBidRequest_NoDeviceOS(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := &openrtb2.BidRequest{
		ID: "no-os-test",
		Imp: []openrtb2.Imp{
			{
				DisplayManager: "adikteev",
				Banner: &openrtb2.Banner{
					W: ptrInt64(320),
					H: ptrInt64(50),
				},
			},
		},
	}

	resp := svc.HandleBidRequest(req)
	if resp != nil {
		t.Fatal("expected nil response when device OS is empty (key includes empty string)")
	}
}

func TestHandleBidRequest_EmptyImpArray(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := &openrtb2.BidRequest{
		ID:  "empty-imp-test",
		Imp: []openrtb2.Imp{},
	}

	if resp := svc.HandleBidRequest(req); resp != nil {
		t.Error("expected nil response for empty imp array")
	}
}

func TestHandleBidRequest_UnknownDisplayManager(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := &openrtb2.BidRequest{
		ID: "unknown-dm-test",
		Device: &openrtb2.Device{
			OS: "android",
		},
		Imp: []openrtb2.Imp{
			{
				DisplayManager: "unknown",
				Banner: &openrtb2.Banner{
					W: ptrInt64(320),
					H: ptrInt64(50),
				},
			},
		},
	}

	if resp := svc.HandleBidRequest(req); resp != nil {
		t.Error("expected nil response for unknown displaymanager")
	}
}

func TestHandleBidRequest_MissingDims(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := &openrtb2.BidRequest{
		ID: "missing-dims-test",
		Device: &openrtb2.Device{
			OS: "android",
		},
		Imp: []openrtb2.Imp{
			{
				DisplayManager: "adikteev",
				Banner: &openrtb2.Banner{
					W: nil,
					H: nil,
				},
			},
		},
	}

	if resp := svc.HandleBidRequest(req); resp != nil {
		t.Error("expected nil response for missing dimensions (0x0 key has no match)")
	}
}

func TestHandleBidRequest_MultiImp(t *testing.T) {
	store, _ := dspsimulator.NewResponseStore(testdataDir)
	svc := dspsimulator.NewService(store)

	req := &openrtb2.BidRequest{
		ID: "multi-imp-test",
		Device: &openrtb2.Device{
			OS: "android",
		},
		Imp: []openrtb2.Imp{
			{
				DisplayManager: "adikteev",
				Banner: &openrtb2.Banner{
					W: ptrInt64(320),
					H: ptrInt64(50),
				},
			},
			{
				DisplayManager: "adikteev",
				Video: &openrtb2.Video{
					W:     360,
					H:     640,
					MIMEs: []string{"video/mp4"},
				},
			},
		},
	}

	resp := svc.HandleBidRequest(req)
	if resp == nil {
		t.Fatal("expected non-nil response for multi-imp")
	}

	if len(resp.SeatBid) < 2 {
		for i, sb := range resp.SeatBid {
			for j, b := range sb.Bid {
				t.Logf("seatbid[%d].bid[%d]: impid=%s price=%f", i, j, b.ImpID, b.Price)
			}
		}
		t.Errorf("expected at least 2 seatbids for 2 imps, got %d", len(resp.SeatBid))
	}

	fmt.Printf("multi-imp response: %d seatbids\n", len(resp.SeatBid))
}
