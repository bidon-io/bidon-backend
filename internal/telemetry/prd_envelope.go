package telemetry

import (
	"strings"

	"github.com/bidon-io/bidon-backend/internal/device"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

// zeroAdvertisingID is the all-zero UUID the PRD requires whenever the real
// advertising id must not be forwarded. It is always written, so the field is
// present on the wire.
const zeroAdvertisingID = "00000000-0000-0000-0000-000000000000"

// applyPRDEnvelope fills the PRD common header from the auction request.
// Fields the request does not carry stay at the proto3 default.
// country is left as the server MaxMind value already on the envelope.
func applyPRDEnvelope(env *telemetryv1.Envelope, base *schema.BaseRequest) {
	if env == nil {
		return
	}
	id, state := advertisingID(base)
	env.AdvertisingId = id
	env.AdvertisingIdState = state
	if base == nil {
		return
	}

	env.Host, env.AuctionOwner, env.IntegrationMode, env.Mediator = integration(base)
	if env.Bundle == "" {
		env.Bundle = base.App.Bundle
	}
	if env.AppKey == "" {
		env.AppKey = base.App.Key
	}
	env.SdkVersion = base.App.SDKVersion
	env.AppVersion = base.App.Version
	env.Os = base.Device.OS
	env.OsVersion = base.Device.OSVersion
	env.DeviceModel = base.Device.Model
	env.DeviceManufacturer = base.Device.Manufacturer
	env.ConnectionType = connectionType(base.Device.ConnectionType)
	env.ScreenWidth = int32(base.Device.Width)
	env.ScreenHeight = int32(base.Device.Height)
	env.ScreenDensity = base.Device.PXRatio
	env.DeviceType = openRTBDeviceType(base.Device.Type)
	if base.Session.StartTS != 0 {
		env.SessionStartTs = int64(base.Session.StartTS)
	}

	regs := base.GetRegulations()
	env.GdprApplies = regs.GDPR
	env.UsPrivacy = regs.USPrivacy
	env.Coppa = coppaFlagged(base)
	env.Lmt = limitAdTracking(base)
}

// advertisingID applies the PRD table. First match wins: COPPA, then consent
// absent, then limit-ad-tracking or a deleted id, otherwise the real id.
// Without a request the id is the zero UUID and the state stays unspecified,
// because no consent signal was evaluated.
func advertisingID(base *schema.BaseRequest) (string, telemetryv1.AdvertisingIdState) {
	if base == nil {
		return zeroAdvertisingID, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_UNSPECIFIED
	}
	if coppaFlagged(base) {
		return zeroAdvertisingID, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_COPPA
	}
	if !trackingAuthorized(base) {
		return zeroAdvertisingID, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_NO_CONSENT
	}
	id := strings.ToLower(strings.TrimSpace(base.User.IDFA))
	if id == "" || id == zeroAdvertisingID {
		return zeroAdvertisingID, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_LAT
	}
	return id, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_REAL
}

func coppaFlagged(base *schema.BaseRequest) bool {
	if base.GetRegulations().COPPA {
		return true
	}
	return base.User.COPPA != nil && *base.User.COPPA
}

func trackingAuthorized(base *schema.BaseRequest) bool {
	return strings.EqualFold(base.User.TrackingAuthorizationStatus, "authorized")
}

func limitAdTracking(base *schema.BaseRequest) bool {
	if !trackingAuthorized(base) {
		return true
	}
	id := strings.ToLower(strings.TrimSpace(base.User.IDFA))
	return id == "" || id == zeroAdvertisingID
}

// integration labels the current server path. /v2/auction is Bidon running
// the auction, so auction_owner is bidon. max_bidding is recorded when the
// request says so; this process still owns the auction.
func integration(base *schema.BaseRequest) (telemetryv1.Host, telemetryv1.AuctionOwner, telemetryv1.IntegrationMode, telemetryv1.Mediator) {
	mode := strings.ToLower(strings.TrimSpace(base.GetMediationMode()))
	mediator := strings.ToLower(strings.TrimSpace(base.GetMediator()))
	if mode == "" && mediator == "max" {
		mode = "max_custom"
	}
	owner := telemetryv1.AuctionOwner_AUCTION_OWNER_BIDON
	switch mode {
	case "max", "max_custom":
		return telemetryv1.Host_HOST_MAX_ADAPTER, owner, telemetryv1.IntegrationMode_INTEGRATION_MODE_MAX_CUSTOM, telemetryv1.Mediator_MEDIATOR_MAX
	case "max_bidding":
		return telemetryv1.Host_HOST_MAX_ADAPTER, owner, telemetryv1.IntegrationMode_INTEGRATION_MODE_MAX_BIDDING, telemetryv1.Mediator_MEDIATOR_MAX
	default:
		return telemetryv1.Host_HOST_PUBLISHER, owner, telemetryv1.IntegrationMode_INTEGRATION_MODE_DIRECT, mediatorFrom(mediator)
	}
}

func mediatorFrom(mediator string) telemetryv1.Mediator {
	switch mediator {
	case "", "bidon":
		return telemetryv1.Mediator_MEDIATOR_BIDON
	case "max":
		return telemetryv1.Mediator_MEDIATOR_MAX
	case "level_play", "levelplay":
		return telemetryv1.Mediator_MEDIATOR_LEVEL_PLAY
	case "admob":
		return telemetryv1.Mediator_MEDIATOR_ADMOB
	default:
		return telemetryv1.Mediator_MEDIATOR_UNSPECIFIED
	}
}

func connectionType(raw string) telemetryv1.ConnectionType {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "":
		return telemetryv1.ConnectionType_CONNECTION_TYPE_UNSPECIFIED
	case "WIFI":
		return telemetryv1.ConnectionType_CONNECTION_TYPE_WIFI
	case "CELLULAR_4_G":
		return telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR_4G
	case "CELLULAR_5_G":
		return telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR_5G
	case "ETHERNET":
		return telemetryv1.ConnectionType_CONNECTION_TYPE_ETHERNET
	case "CELLULAR":
		return telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR
	case "CELLULAR_2_G":
		return telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR_2G
	case "CELLULAR_3_G":
		return telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR_3G
	default:
		return telemetryv1.ConnectionType_CONNECTION_TYPE_UNKNOWN
	}
}

func openRTBDeviceType(t device.Type) int32 {
	switch t {
	case device.PhoneType:
		return 4
	case device.TabletType:
		return 5
	default:
		return 0
	}
}
