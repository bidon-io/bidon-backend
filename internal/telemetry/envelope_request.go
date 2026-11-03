package telemetry

import (
	"strings"

	"github.com/prebid/openrtb/v19/adcom1"

	"github.com/bidon-io/bidon-backend/internal/device"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

// zeroAdvertisingID is the all-zero UUID written whenever the real advertising
// id must not be forwarded. It is always set so the field is present on the wire.
const zeroAdvertisingID = "00000000-0000-0000-0000-000000000000"

// fillEnvelopeFromRequest fills the envelope groups from the auction request.
// Fields the request does not carry stay at the proto3 default.
// country stays the server MaxMind value already on the envelope.
func fillEnvelopeFromRequest(env *telemetryv1.Envelope, req *schema.BaseRequest) {
	if env == nil {
		return
	}
	id, state, lmt := advertisingID(req)
	env.Privacy = &telemetryv1.Envelope_Privacy{
		AdvertisingId:      id,
		AdvertisingIdState: state,
		Lmt:                lmt,
	}
	if req == nil {
		return
	}

	host, mode, mediator := integration(req)
	env.Integration = &telemetryv1.Envelope_Integration{
		Host: host,
		// This server runs the auction, including when MAX embeds the SDK.
		AuctionOwner:    telemetryv1.AuctionOwner_AUCTION_OWNER_BIDON,
		IntegrationMode: mode,
		Mediator:        mediator,
	}

	env.App = &telemetryv1.Envelope_App{
		AppKey:     req.App.Key,
		Bundle:     req.App.Bundle,
		AppVersion: req.App.Version,
		SdkVersion: req.App.SDKVersion,
	}

	env.Device = &telemetryv1.Envelope_Device{
		Os:             req.Device.OS,
		OsVersion:      req.Device.OSVersion,
		Model:          req.Device.Model,
		Manufacturer:   req.Device.Manufacturer,
		DeviceType:     deviceType(req.Device.Type),
		ConnectionType: connectionType(req.Device.ConnectionType),
		ScreenWidth:    int32(req.Device.Width),
		ScreenHeight:   int32(req.Device.Height),
		ScreenDensity:  req.Device.PXRatio,
	}

	env.Session = &telemetryv1.Envelope_Session{StartTs: int64(req.Session.StartTS)}

	regs := req.GetRegulations()
	env.Privacy.Coppa = coppaFlagged(req)
	env.Privacy.GdprApplies = regs.GDPR
	env.Privacy.UsPrivacy = regs.USPrivacy
	env.Privacy.HasTcfString = regs.EUPrivacy != ""
}

// deviceType is the OpenRTB device type: 4 phone, 5 tablet.
func deviceType(t device.Type) int32 {
	switch t {
	case device.PhoneType:
		return int32(adcom1.DevicePhone)
	case device.TabletType:
		return int32(adcom1.DeviceTablet)
	}
	return 0
}

// advertisingID applies the order: COPPA, then missing consent, then
// limit-ad-tracking or a deleted id, otherwise the real id. lmt follows the
// request, so COPPA can zero the id while lmt stays false. With no request
// the id is the zero UUID, the state stays unspecified, and lmt is false.
//
// "Consent" here is only the request's tracking_authorization_status (the iOS
// tracking permission). GDPR, TCF and GPP consent are not evaluated yet.
// TODO: derive consent across platforms and from the regulation strings.
func advertisingID(req *schema.BaseRequest) (string, telemetryv1.AdvertisingIdState, bool) {
	if req == nil {
		return zeroAdvertisingID, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_UNSPECIFIED, false
	}
	raw := strings.ToLower(strings.TrimSpace(req.User.IDFA))
	authorized := strings.EqualFold(req.User.TrackingAuthorizationStatus, "authorized")
	isReal := raw != "" && raw != zeroAdvertisingID
	lmt := !authorized || !isReal

	switch {
	case coppaFlagged(req):
		return zeroAdvertisingID, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_COPPA, lmt
	case !authorized:
		return zeroAdvertisingID, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_NO_CONSENT, lmt
	case !isReal:
		return zeroAdvertisingID, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_LAT, lmt
	default:
		return raw, telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_REAL, lmt
	}
}

// coppaFlagged is true when either the regulations or the user flag COPPA. The
// OpenRTB builder sends only regs.COPPA, so telemetry can flag COPPA on an
// auction whose bid requests did not.
func coppaFlagged(req *schema.BaseRequest) bool {
	return req.GetRegulations().COPPA || (req.User.COPPA != nil && *req.User.COPPA)
}

// integration labels the SDK host. A blank mode with mediator max is the MAX
// custom adapter. max_bidding is recorded when the request says so.
func integration(req *schema.BaseRequest) (telemetryv1.Host, telemetryv1.IntegrationMode, telemetryv1.Mediator) {
	mode := strings.ToLower(strings.TrimSpace(req.GetMediationMode()))
	mediator := strings.ToLower(strings.TrimSpace(req.GetMediator()))
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
	case "CELLULAR", "CELLULAR_UNKNOWN":
		return telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR
	case "CELLULAR_2_G":
		return telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR_2G
	case "CELLULAR_3_G":
		return telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR_3G
	default:
		return telemetryv1.ConnectionType_CONNECTION_TYPE_UNKNOWN
	}
}
