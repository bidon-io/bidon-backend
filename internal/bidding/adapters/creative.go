package adapters

import (
	"encoding/json"

	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

// Common video MIME lists used by OpenRTB dual / fullscreen creative shapes.
var (
	InterstitialVideoMIMEs = []string{
		"video/mp4", "video/3gpp", "video/3gpp2", "video/x-m4v", "video/quicktime",
	}
	RewardedVideoMIMEs = []string{
		"video/mp4", "video/x-m4v", "video/quicktime", "video/mpeg", "video/avi",
	}
	AllVideoProtocols = []adcom1.MediaCreativeSubtype{
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
	}
)

// ResolveFullscreenSize returns WxH from FullscreenFormats for the device type.
// Unknown device types fall back to PHONE. When swapOrientation is true and the
// request is landscape, width and height are swapped.
func ResolveFullscreenSize(deviceType string, portrait bool, swapOrientation bool) (w, h int64) {
	size, ok := FullscreenFormats[deviceType]
	if !ok {
		size = FullscreenFormats["PHONE"]
	}
	w, h = size[0], size[1]
	if swapOrientation && !portrait {
		w, h = h, w
	}
	return w, h
}

// BannerImpOptions configures BuildBannerImp.
type BannerImpOptions struct {
	Size BannerSizeOptions

	// API sets Banner.API (e.g. MRAID frameworks). Omitted when empty.
	API []adcom1.APIFramework

	// Width / Height override the resolved size when non-nil (e.g. Meta adaptive phone W=0).
	Width  *int64
	Height *int64
}

// BuildBannerImp builds a creative-only banner Imp (Instl=0, AboveFold, sized Banner).
// Shell fields (TagID, bidfloor, display manager) stay in BuildRTBRequest.
func BuildBannerImp(auctionRequest *schema.AuctionRequest, opts BannerImpOptions) (*openrtb2.Imp, error) {
	size, err := ResolveBannerSize(
		auctionRequest.AdObject.Format(),
		auctionRequest.AdObject.IsAdaptive(),
		auctionRequest.Device.IsTablet(),
		opts.Size,
	)
	if err != nil {
		return nil, err
	}

	w, h := size[0], size[1]
	if opts.Width != nil {
		w = *opts.Width
	}
	if opts.Height != nil {
		h = *opts.Height
	}

	banner := &openrtb2.Banner{
		W:   &w,
		H:   &h,
		Pos: adcom1.PositionAboveFold.Ptr(),
	}
	if len(opts.API) > 0 {
		banner.API = opts.API
	}

	return &openrtb2.Imp{
		Instl:  0,
		Banner: banner,
	}, nil
}

// InterstitialImpOptions configures BuildInterstitialImp.
type InterstitialImpOptions struct {
	// DisableOrientationSwap keeps portrait FullscreenFormats dimensions even for landscape.
	// Default (false) swaps WxH for landscape, matching most adapters.
	DisableOrientationSwap bool

	// IncludeVideo adds a dual Banner+Video creative (moloco/startio/bidmachine).
	IncludeVideo bool

	// VideoMIMEs overrides InterstitialVideoMIMEs when IncludeVideo is set.
	VideoMIMEs []string

	// BannerAPI sets Banner.API when non-empty.
	BannerAPI []adcom1.APIFramework
}

// BuildInterstitialImp builds a creative-only interstitial Imp (Instl=1, fullscreen Banner).
// With IncludeVideo, adds empty Banner BType/BAttr and a fullscreen Video object.
func BuildInterstitialImp(auctionRequest *schema.AuctionRequest, opts InterstitialImpOptions) *openrtb2.Imp {
	w, h := ResolveFullscreenSize(
		string(auctionRequest.Device.Type),
		auctionRequest.AdObject.IsPortrait(),
		!opts.DisableOrientationSwap,
	)

	banner := &openrtb2.Banner{
		W:   &w,
		H:   &h,
		Pos: adcom1.PositionFullScreen.Ptr(),
	}
	if opts.IncludeVideo {
		banner.BType = []openrtb2.BannerAdType{}
		banner.BAttr = []adcom1.CreativeAttribute{}
	}
	if len(opts.BannerAPI) > 0 {
		banner.API = opts.BannerAPI
	}

	imp := &openrtb2.Imp{
		Instl:  1,
		Banner: banner,
	}

	if opts.IncludeVideo {
		mimes := opts.VideoMIMEs
		if len(mimes) == 0 {
			mimes = append([]string(nil), InterstitialVideoMIMEs...)
		}
		imp.Video = &openrtb2.Video{
			W:     w,
			H:     h,
			Pos:   adcom1.PositionFullScreen.Ptr(),
			MIMEs: mimes,
		}
	}

	return imp
}

// RewardedImpOptions configures BuildRewardedImp for common dual / video-fullscreen shapes.
// Divergent rewarded creatives (network Ext-only, fixed 1920×1080, rich VAST, etc.) stay custom.
type RewardedImpOptions struct {
	// DisableOrientationSwap keeps portrait FullscreenFormats dimensions even for landscape.
	DisableOrientationSwap bool

	// IncludeBanner adds a dual Banner+Video creative (moloco/startio/bidmachine).
	IncludeBanner bool

	// Rwdd sets Imp.Rwdd when non-zero.
	Rwdd int8

	// Skip sets Video.Skip when non-nil.
	Skip *int8

	// VideoMIMEs overrides RewardedVideoMIMEs when non-empty.
	VideoMIMEs []string

	// Protocols overrides AllVideoProtocols when non-empty.
	Protocols []adcom1.MediaCreativeSubtype

	BannerBAttr []adcom1.CreativeAttribute
	VideoBAttr  []adcom1.CreativeAttribute

	ImpExt   json.RawMessage
	VideoExt json.RawMessage
}

// BuildRewardedImp builds a creative-only rewarded Imp (Instl=1, fullscreen Video).
// Optional Banner, Rwdd, Skip, BAttr, and Ext cover the common dual rewarded shape.
func BuildRewardedImp(auctionRequest *schema.AuctionRequest, opts RewardedImpOptions) *openrtb2.Imp {
	w, h := ResolveFullscreenSize(
		string(auctionRequest.Device.Type),
		auctionRequest.AdObject.IsPortrait(),
		!opts.DisableOrientationSwap,
	)

	imp := &openrtb2.Imp{
		Instl: 1,
	}
	if opts.Rwdd != 0 {
		imp.Rwdd = opts.Rwdd
	}
	if len(opts.ImpExt) > 0 {
		imp.Ext = opts.ImpExt
	}

	if opts.IncludeBanner {
		banner := &openrtb2.Banner{
			W:     &w,
			H:     &h,
			BType: []openrtb2.BannerAdType{},
			Pos:   adcom1.PositionFullScreen.Ptr(),
		}
		if opts.BannerBAttr != nil {
			banner.BAttr = opts.BannerBAttr
		}
		imp.Banner = banner
	}

	mimes := opts.VideoMIMEs
	if len(mimes) == 0 {
		mimes = append([]string(nil), RewardedVideoMIMEs...)
	}
	protocols := opts.Protocols
	if len(protocols) == 0 {
		protocols = append([]adcom1.MediaCreativeSubtype(nil), AllVideoProtocols...)
	}

	video := &openrtb2.Video{
		W:         w,
		H:         h,
		Pos:       adcom1.PositionFullScreen.Ptr(),
		MIMEs:     mimes,
		Protocols: protocols,
	}
	if opts.VideoBAttr != nil {
		video.BAttr = opts.VideoBAttr
	}
	if opts.Skip != nil {
		video.Skip = opts.Skip
	}
	if len(opts.VideoExt) > 0 {
		video.Ext = opts.VideoExt
	}
	imp.Video = video

	return imp
}
