package rendering

// Config is the DSP-controlled rendering configuration carried in bid.ext.rendering.
// When rendering is omitted entirely, documented defaults are applied. When the DSP
// provides rendering, creasty/defaults fills missing fields and go-playground/validator
// checks constraints. Invalid configs fall back to defaults because OpenRTB cannot
// reject or correct a bid for rendering problems.
type Config struct {
	CloseButton *CloseButtonConfig `json:"close_button,omitempty"`
	EndCards    *EndCardsConfig    `json:"endcards,omitempty"`
	Container   *ContainerConfig   `json:"container,omitempty"`
	Creative    *CreativeConfig    `json:"creative,omitempty"`
	StoreKit    *StoreKitConfig    `json:"store_kit,omitempty"`
}

type CloseButtonStyle string

const (
	CloseButtonStyleIconX      CloseButtonStyle = "icon_x"
	CloseButtonStyleIconCircle CloseButtonStyle = "icon_circle"
	CloseButtonStyleTextClose  CloseButtonStyle = "text_close"
	CloseButtonStyleCustom     CloseButtonStyle = "custom"
)

type CloseButtonPosition string

const (
	CloseButtonPositionTopLeft     CloseButtonPosition = "top_left"
	CloseButtonPositionTopRight    CloseButtonPosition = "top_right"
	CloseButtonPositionBottomLeft  CloseButtonPosition = "bottom_left"
	CloseButtonPositionBottomRight CloseButtonPosition = "bottom_right"
)

type CloseButtonConfig struct {
	Visible          *bool               `json:"visible,omitempty" default:"true"`
	DelaySeconds     *float64            `json:"delay_seconds,omitempty" default:"5"`
	CountdownVisible *bool               `json:"countdown_visible,omitempty" default:"true"`
	Style            CloseButtonStyle    `json:"style,omitempty" default:"icon_x" validate:"omitempty,oneof=icon_x icon_circle text_close custom"`
	SizeDP           *float64            `json:"size_dp,omitempty" default:"30"`
	Position         CloseButtonPosition `json:"position,omitempty" default:"top_right" validate:"omitempty,oneof=top_left top_right bottom_left bottom_right"`
	Color            string              `json:"color,omitempty" default:"#FFFFFF" validate:"omitempty,hexcolor"`
	Opacity          *float64            `json:"opacity,omitempty" default:"0.9" validate:"omitempty,gte=0,lte=1"`
	PaddingDP        *float64            `json:"padding_dp,omitempty" default:"12" validate:"omitempty,gte=0"`
	CustomAssetURL   string              `json:"custom_asset_url,omitempty" validate:"custom_asset_url"`
}

type EndCardsLayout string

const (
	EndCardsLayoutSingle   EndCardsLayout = "single"
	EndCardsLayoutCarousel EndCardsLayout = "carousel"
	EndCardsLayoutGrid     EndCardsLayout = "grid"
)

type EndCardsConfig struct {
	Enabled                *bool          `json:"enabled,omitempty" default:"false"`
	Count                  *int           `json:"count,omitempty" default:"1" validate:"omitempty,gte=1,lte=3"`
	Layout                 EndCardsLayout `json:"layout,omitempty" default:"single" validate:"omitempty,oneof=single carousel grid"`
	AutoAdvanceSeconds     *float64       `json:"auto_advance_seconds,omitempty" default:"0" validate:"omitempty,gte=0"`
	CTAText                string         `json:"cta_text,omitempty" default:"Install Now"`
	CTAColor               string         `json:"cta_color,omitempty" default:"#007AFF" validate:"omitempty,hexcolor"`
	Assets                 []EndcardAsset `json:"assets,omitempty" default:"[]" validate:"omitempty,dive"`
	DismissOnBackgroundTap *bool          `json:"dismiss_on_background_tap,omitempty" default:"true"`
}

type EndcardAsset struct {
	URL string `json:"url" validate:"required,httpurl"`
}

type ContainerFormat string

const (
	ContainerFormatBanner       ContainerFormat = "banner"
	ContainerFormatInterstitial ContainerFormat = "interstitial"
	ContainerFormatRewarded     ContainerFormat = "rewarded"
	ContainerFormatNative       ContainerFormat = "native"
	ContainerFormatMREC         ContainerFormat = "mrec"
)

type ContainerOrientation string

const (
	ContainerOrientationPortrait   ContainerOrientation = "portrait"
	ContainerOrientationLandscape  ContainerOrientation = "landscape"
	ContainerOrientationResponsive ContainerOrientation = "responsive"
)

type ContainerConfig struct {
	Format             ContainerFormat           `json:"format,omitempty" default:"interstitial" validate:"omitempty,oneof=banner interstitial rewarded native mrec"`
	Orientation        ContainerOrientation      `json:"orientation,omitempty" default:"responsive" validate:"omitempty,oneof=portrait landscape responsive"`
	BackgroundColor    string                    `json:"background_color,omitempty" default:"#000000" validate:"omitempty,hexcolor"`
	BackgroundBlur     *bool                     `json:"background_blur,omitempty" default:"false"`
	MaxDurationSeconds *float64                  `json:"max_duration_seconds,omitempty" default:"30" validate:"omitempty,gte=0"`
	SkipOffsetSeconds  *float64                  `json:"skip_offset_seconds,omitempty" default:"15" validate:"omitempty,gte=0"`
	MuteOnStart        *bool                     `json:"mute_on_start,omitempty" default:"true"`
	ImpressionTracking *ImpressionTrackingConfig `json:"impression_tracking,omitempty"`
}

type ImpressionTrackingConfig struct {
	MinViewablePct     *float64 `json:"min_viewable_pct,omitempty" default:"50" validate:"omitempty,gte=0,lte=100"`
	MinViewableSeconds *float64 `json:"min_viewable_seconds,omitempty" default:"1" validate:"omitempty,gte=0"`
}

type CreativeType string

const (
	CreativeTypeVAST        CreativeType = "vast"
	CreativeTypeMRAID       CreativeType = "mraid"
	CreativeTypeHTML        CreativeType = "html"
	CreativeTypeStaticImage CreativeType = "static_image"
	CreativeTypeNative      CreativeType = "native"
	CreativeTypePlayable    CreativeType = "playable"
)

type CreativeSource string

const (
	CreativeSourceADM  CreativeSource = "seatbid.bid.adm"
	CreativeSourceNUrl CreativeSource = "seatbid.bid.nurl"
)

type CreateMRAIDVersion string

const (
	CreateMRaidVersionV2 CreateMRAIDVersion = "2.0"
	CreateMRaidVersionV3 CreateMRAIDVersion = "3.0"
)

type CreativeVASTVersion string

const (
	CreativeVASTVersionV3  CreativeVASTVersion = "3.0"
	CreativeVASTVersionV4  CreativeVASTVersion = "4.0"
	CreativeVASTVersionV41 CreativeVASTVersion = "4.1"
	CreativeVASTVersionV42 CreativeVASTVersion = "4.2"
)

type CreativePreloadStrategy string

const (
	CreativePreloadStrategyEager    CreativePreloadStrategy = "eager"
	CreativePreloadStrategyLazy     CreativePreloadStrategy = "lazy"
	CreativePreloadStrategyOnDemand CreativePreloadStrategy = "on_demand"
)

type CreativeConfig struct {
	Type              CreativeType            `json:"type,omitempty" default:"static_image" validate:"required,oneof=mraid vast html static_image native playable"`
	Source            CreativeSource          `json:"source,omitempty" default:"seatbid.bid.adm" validate:"omitempty,oneof=seatbid.bid.adm seatbid.bid.nurl"`
	MRAIDVersion      CreateMRAIDVersion      `json:"mraid_version,omitempty" default:"3.0" validate:"omitempty,oneof=2.0 3.0"`
	VASTVersion       CreativeVASTVersion     `json:"vast_version,omitempty" default:"4.2" validate:"omitempty,oneof=3.0 4.0 4.1 4.2"`
	VPAIDEnabled      *bool                   `json:"vpaid_enabled,omitempty" default:"false"`
	HTMLSandboxPolicy string                  `json:"html_sandbox_policy,omitempty" default:"allow-scripts"`
	PreloadStrategy   CreativePreloadStrategy `json:"preload_strategy,omitempty" default:"eager" validate:"omitempty,oneof=eager lazy on_demand"`
}

type StoreKitTrigger string

const (
	StoreKitTriggerOnImpression StoreKitTrigger = "on_impression"
	StoreKitTriggerOnEndCard    StoreKitTrigger = "on_endcard"
	StoreKitTriggerOnClick      StoreKitTrigger = "on_click"
)

type StoreKitPosition string

const (
	StoreKitPositionBottom  StoreKitPosition = "bottom"
	StoreKitPositionOverlay StoreKitPosition = "overlay"
)

type StoreKitConfig struct {
	Enabled         *bool            `json:"enabled,omitempty" default:"false"`
	AppStoreID      string           `json:"app_store_id,omitempty" validate:"required_if=Enabled true"`
	Trigger         StoreKitTrigger  `json:"trigger,omitempty" default:"on_endcard" validate:"omitempty,oneof=on_impression on_endcard on_click"`
	DelaySeconds    *float64         `json:"delay_seconds,omitempty" default:"0" validate:"omitempty,gte=0"`
	UserDismissable *bool            `json:"user_dismissable,omitempty" default:"false"`
	DismissOnClose  *bool            `json:"dismiss_on_close,omitempty" default:"true"`
	Position        StoreKitPosition `json:"position,omitempty" default:"bottom" validate:"omitempty,oneof=bottom overlay"`
	CampaignToken   string           `json:"campaign_token,omitempty"`
}
