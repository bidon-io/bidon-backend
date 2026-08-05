package adapters_test

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v19/adcom1"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/device"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

func TestResolveFullscreenSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		deviceType      string
		portrait        bool
		swapOrientation bool
		wantW           int64
		wantH           int64
	}{
		{
			name:            "phone portrait",
			deviceType:      "PHONE",
			portrait:        true,
			swapOrientation: true,
			wantW:           320,
			wantH:           480,
		},
		{
			name:            "phone landscape swaps",
			deviceType:      "PHONE",
			portrait:        false,
			swapOrientation: true,
			wantW:           480,
			wantH:           320,
		},
		{
			name:            "phone landscape without swap",
			deviceType:      "PHONE",
			portrait:        false,
			swapOrientation: false,
			wantW:           320,
			wantH:           480,
		},
		{
			name:            "tablet portrait",
			deviceType:      "TABLET",
			portrait:        true,
			swapOrientation: true,
			wantW:           768,
			wantH:           1024,
		},
		{
			name:            "unknown device falls back to phone",
			deviceType:      "UNKNOWN",
			portrait:        true,
			swapOrientation: true,
			wantW:           320,
			wantH:           480,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w, h := adapters.ResolveFullscreenSize(tt.deviceType, tt.portrait, tt.swapOrientation)
			if w != tt.wantW || h != tt.wantH {
				t.Fatalf("size = %dx%d, want %dx%d", w, h, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestBuildBannerImp(t *testing.T) {
	t.Parallel()

	auctionRequest := bannerAuctionRequest(ad.BannerFormat, false)

	imp, err := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if imp.Instl != 0 {
		t.Fatalf("Instl = %d, want 0", imp.Instl)
	}
	if imp.Banner == nil {
		t.Fatal("expected Banner")
	}
	if imp.Banner.W == nil || *imp.Banner.W != 320 || imp.Banner.H == nil || *imp.Banner.H != 50 {
		t.Fatalf("Banner size = %v x %v, want 320x50", imp.Banner.W, imp.Banner.H)
	}
	if imp.Banner.Pos == nil || *imp.Banner.Pos != adcom1.PositionAboveFold {
		t.Fatalf("Pos = %v, want AboveFold", imp.Banner.Pos)
	}
	if imp.TagID != "" || imp.DisplayManager != "" || imp.BidFloor != 0 {
		t.Fatalf("shell fields must stay unset, got TagID=%q DM=%q floor=%v", imp.TagID, imp.DisplayManager, imp.BidFloor)
	}
}

func TestBuildBannerImp_APIAndOverrides(t *testing.T) {
	t.Parallel()

	auctionRequest := bannerAuctionRequest(ad.AdaptiveFormat, false)
	w, h := int64(0), int64(50)

	imp, err := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{
		API:    []adcom1.APIFramework{3, 5},
		Width:  &w,
		Height: &h,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *imp.Banner.W != 0 || *imp.Banner.H != 50 {
		t.Fatalf("Banner size = %dx%d, want 0x50", *imp.Banner.W, *imp.Banner.H)
	}
	if len(imp.Banner.API) != 2 || imp.Banner.API[0] != 3 || imp.Banner.API[1] != 5 {
		t.Fatalf("API = %v, want [3 5]", imp.Banner.API)
	}
}

func TestBuildBannerImp_AdaptiveTablet(t *testing.T) {
	t.Parallel()

	auctionRequest := bannerAuctionRequest(ad.AdaptiveFormat, true)
	imp, err := adapters.BuildBannerImp(auctionRequest, adapters.BannerImpOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *imp.Banner.W != 728 || *imp.Banner.H != 90 {
		t.Fatalf("Banner size = %dx%d, want 728x90", *imp.Banner.W, *imp.Banner.H)
	}
}

func TestBuildInterstitialImp_BannerOnly(t *testing.T) {
	t.Parallel()

	auctionRequest := fullscreenAuctionRequest(true, device.PhoneType)
	imp := adapters.BuildInterstitialImp(auctionRequest, adapters.InterstitialImpOptions{})

	if imp.Instl != 1 {
		t.Fatalf("Instl = %d, want 1", imp.Instl)
	}
	if imp.Banner == nil || imp.Video != nil {
		t.Fatalf("expected banner-only, got banner=%v video=%v", imp.Banner != nil, imp.Video != nil)
	}
	if *imp.Banner.W != 320 || *imp.Banner.H != 480 {
		t.Fatalf("Banner size = %dx%d, want 320x480", *imp.Banner.W, *imp.Banner.H)
	}
	if imp.Banner.Pos == nil || *imp.Banner.Pos != adcom1.PositionFullScreen {
		t.Fatalf("Pos = %v, want FullScreen", imp.Banner.Pos)
	}
}

func TestBuildInterstitialImp_LandscapeSwapAndDualVideo(t *testing.T) {
	t.Parallel()

	auctionRequest := fullscreenAuctionRequest(false, device.PhoneType)
	imp := adapters.BuildInterstitialImp(auctionRequest, adapters.InterstitialImpOptions{
		IncludeVideo: true,
		BannerAPI:    []adcom1.APIFramework{3, 5},
	})

	if *imp.Banner.W != 480 || *imp.Banner.H != 320 {
		t.Fatalf("Banner size = %dx%d, want 480x320", *imp.Banner.W, *imp.Banner.H)
	}
	if imp.Banner.BType == nil || imp.Banner.BAttr == nil {
		t.Fatal("expected empty BType/BAttr slices for dual creative")
	}
	if len(imp.Banner.API) != 2 {
		t.Fatalf("API = %v, want [3 5]", imp.Banner.API)
	}
	if imp.Video == nil {
		t.Fatal("expected Video")
	}
	if imp.Video.W != 480 || imp.Video.H != 320 {
		t.Fatalf("Video size = %dx%d, want 480x320", imp.Video.W, imp.Video.H)
	}
	if len(imp.Video.MIMEs) != len(adapters.InterstitialVideoMIMEs) {
		t.Fatalf("MIMEs = %v, want defaults", imp.Video.MIMEs)
	}
}

func TestBuildInterstitialImp_DisableOrientationSwap(t *testing.T) {
	t.Parallel()

	auctionRequest := fullscreenAuctionRequest(false, device.PhoneType)
	imp := adapters.BuildInterstitialImp(auctionRequest, adapters.InterstitialImpOptions{
		DisableOrientationSwap: true,
	})

	if *imp.Banner.W != 320 || *imp.Banner.H != 480 {
		t.Fatalf("Banner size = %dx%d, want 320x480", *imp.Banner.W, *imp.Banner.H)
	}
}

func TestBuildRewardedImp_Dual(t *testing.T) {
	t.Parallel()

	auctionRequest := fullscreenAuctionRequest(true, device.PhoneType)
	skip := int8(1)
	imp := adapters.BuildRewardedImp(auctionRequest, adapters.RewardedImpOptions{
		IncludeBanner: true,
		Rwdd:          1,
		Skip:          &skip,
		BannerBAttr:   []adcom1.CreativeAttribute{16},
		VideoBAttr:    []adcom1.CreativeAttribute{1, 2, 5, 8, 9, 14, 17},
	})

	if imp.Instl != 1 || imp.Rwdd != 1 {
		t.Fatalf("Instl/Rwdd = %d/%d, want 1/1", imp.Instl, imp.Rwdd)
	}
	if imp.Banner == nil || imp.Video == nil {
		t.Fatal("expected dual Banner+Video")
	}
	if *imp.Banner.W != 320 || *imp.Banner.H != 480 {
		t.Fatalf("Banner size = %dx%d, want 320x480", *imp.Banner.W, *imp.Banner.H)
	}
	if imp.Video.Skip == nil || *imp.Video.Skip != 1 {
		t.Fatalf("Skip = %v, want 1", imp.Video.Skip)
	}
	if len(imp.Video.Protocols) != len(adapters.AllVideoProtocols) {
		t.Fatalf("Protocols len = %d, want %d", len(imp.Video.Protocols), len(adapters.AllVideoProtocols))
	}
	if imp.TagID != "" || imp.DisplayManager != "" {
		t.Fatal("shell fields must stay unset")
	}
}

func TestBuildRewardedImp_VideoOnlyWithExt(t *testing.T) {
	t.Parallel()

	auctionRequest := fullscreenAuctionRequest(true, device.TabletType)
	imp := adapters.BuildRewardedImp(auctionRequest, adapters.RewardedImpOptions{
		DisableOrientationSwap: true,
		ImpExt:                 json.RawMessage(`{"rewarded": 1}`),
	})

	if imp.Banner != nil {
		t.Fatal("expected video-only")
	}
	if imp.Video == nil {
		t.Fatal("expected Video")
	}
	if imp.Video.W != 768 || imp.Video.H != 1024 {
		t.Fatalf("Video size = %dx%d, want 768x1024", imp.Video.W, imp.Video.H)
	}
	if string(imp.Ext) != `{"rewarded": 1}` {
		t.Fatalf("Ext = %s, want rewarded flag", imp.Ext)
	}
	if len(imp.Video.MIMEs) != len(adapters.RewardedVideoMIMEs) {
		t.Fatalf("MIMEs = %v, want rewarded defaults", imp.Video.MIMEs)
	}
}

func bannerAuctionRequest(format ad.Format, tablet bool) *schema.AuctionRequest {
	deviceType := device.PhoneType
	if tablet {
		deviceType = device.TabletType
	}
	return &schema.AuctionRequest{
		BaseRequest: schema.BaseRequest{
			Device: schema.Device{Type: deviceType},
		},
		AdObject: schema.AdObject{
			Banner: &schema.BannerAdObject{Format: format},
		},
	}
}

func fullscreenAuctionRequest(portrait bool, deviceType device.Type) *schema.AuctionRequest {
	orientation := "LANDSCAPE"
	if portrait {
		orientation = "PORTRAIT"
	}
	return &schema.AuctionRequest{
		BaseRequest: schema.BaseRequest{
			Device: schema.Device{Type: deviceType},
		},
		AdObject: schema.AdObject{
			Orientation:  orientation,
			Interstitial: &schema.InterstitialAdObject{},
		},
	}
}
