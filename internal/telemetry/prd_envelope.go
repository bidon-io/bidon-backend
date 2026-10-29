package telemetry

import (
	"cmp"
	"strings"

	"github.com/bidon-io/bidon-backend/internal/device"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

// zeroAdvertisingID is the all-zero UUID written whenever the real advertising
// id must not be forwarded. It is always set so the field is present on the wire.
const zeroAdvertisingID = "00000000-0000-0000-0000-000000000000"

// applyPRDEnvelope fills the PRD common header from the auction request.
// Fields the request does not carry stay at the proto3 default.
// country stays the server MaxMind value already on the envelope.
func applyPRDEnvelope(env *telemetryv1.Envelope, base *schema.BaseRequest) {
	if env == nil {
		return
	}
	env.AdvertisingId, env.AdvertisingIdState, env.Lmt = advertisingID(base)
	if base == nil {
		return
	}

	// This server runs the auction, including when MAX embeds the SDK.
	env.AuctionOwner = telemetryv1.AuctionOwner_AUCTION_OWNER_BIDON
	env.Host, env.IntegrationMode, env.Mediator = integration(base)

	env.Bundle = cmp.Or(env.Bundle, base.App.Bundle)
	env.AppKey = cmp.Or(env.AppKey, base.App.Key)
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
	// OpenRTB device type: 4 phone, 5 tablet.
	switch base.Device.Type {
	case device.PhoneType:
		env.DeviceType = 4
	case device.TabletType:
		env.DeviceType = 5
	}
	env.SessionStartTs = int64(base.Session.StartTS)

	regs := base.GetRegulations()
	env.GdprApplies = regs.GDPR
	env.UsPrivacy = regs.USPrivacy
	env.Coppa = coppaFlagged(base)
}

// advertisingID applies the PRD order: COPPA, then missing consent, then
// limit-ad-tracking or a deleted id, otherwise the real id. lmt follows the
// request, so COPPA can zero the id while lmt stays false. With no request
// the id is the zero UUID, the state stays unspecified, and lmt is false.
func advertisingID(base *schema.BaseRequest) (string, telemetryv1.AdvertisingIdState, bool) {
	if base == nil {
		return zeroAdvertisingID, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_UNSPECIFIED, false
	}
	raw := strings.ToLower(strings.TrimSpace(base.User.IDFA))
	authorized := strings.EqualFold(base.User.TrackingAuthorizationStatus, "authorized")
	real := raw != "" && raw != zeroAdvertisingID
	lmt := !authorized || !real

	switch {
	case coppaFlagged(base):
		return zeroAdvertisingID, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_COPPA, lmt
	case !authorized:
		return zeroAdvertisingID, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_NO_CONSENT, lmt
	case !real:
		return zeroAdvertisingID, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_LAT, lmt
	default:
		return raw, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_REAL, lmt
	}
}

func coppaFlagged(base *schema.BaseRequest) bool {
	return base.GetRegulations().COPPA || (base.User.COPPA != nil && *base.User.COPPA)
}

// integration labels the SDK host. A blank mode with mediator max is the MAX
// custom adapter. max_bidding is recorded when the request says so.
func integration(base *schema.BaseRequest) (telemetryv1.Host, telemetryv1.IntegrationMode, telemetryv1.Mediator) {
	mode := strings.ToLower(strings.TrimSpace(base.GetMediationMode()))
	mediator := strings.ToLower(strings.TrimSpace(base.GetMediator()))
	switch {
	case mode == "max_bidding":
		return telemetryv1.Host_HOST_MAX_ADAPTER, telemetryv1.IntegrationMode_INTEGRATION_MODE_MAX_BIDDING, telemetryv1.Mediator_MEDIATOR_MAX
	case mode == "max" || mode == "max_custom" || (mode == "" && mediator == "max"):
		return telemetryv1.Host_HOST_MAX_ADAPTER, telemetryv1.IntegrationMode_INTEGRATION_MODE_MAX_CUSTOM, telemetryv1.Mediator_MEDIATOR_MAX
	default:
		return telemetryv1.Host_HOST_PUBLISHER, telemetryv1.IntegrationMode_INTEGRATION_MODE_DIRECT, mediatorFrom(mediator)
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
