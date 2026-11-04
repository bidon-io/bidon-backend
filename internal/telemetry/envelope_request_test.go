package telemetry

import (
	"testing"

	"github.com/bidon-io/bidon-backend/internal/device"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

func TestAdvertisingIDOrder(t *testing.T) {
	realID := "11111111-1111-1111-1111-111111111111"
	coppa := true
	tests := []struct {
		name  string
		base  *schema.BaseRequest
		want  string
		state telemetryv1.AdvertisingIdState
		lmt   bool
	}{
		{
			name:  "no request",
			want:  zeroAdvertisingID,
			state: telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_UNSPECIFIED,
		},
		{
			name: "coppa wins over a real id",
			base: &schema.BaseRequest{
				User: schema.User{IDFA: realID, TrackingAuthorizationStatus: "authorized", COPPA: &coppa},
			},
			want:  zeroAdvertisingID,
			state: telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_COPPA,
		},
		{
			name: "coppa with no consent still sets lmt",
			base: &schema.BaseRequest{
				User: schema.User{IDFA: realID, TrackingAuthorizationStatus: "denied", COPPA: &coppa},
			},
			want:  zeroAdvertisingID,
			state: telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_COPPA,
			lmt:   true,
		},
		{
			name: "missing consent",
			base: &schema.BaseRequest{
				User: schema.User{IDFA: realID, TrackingAuthorizationStatus: "denied"},
			},
			want:  zeroAdvertisingID,
			state: telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_NO_CONSENT,
			lmt:   true,
		},
		{
			name: "authorized with a deleted id",
			base: &schema.BaseRequest{
				User: schema.User{IDFA: zeroAdvertisingID, TrackingAuthorizationStatus: "AUTHORIZED"},
			},
			want:  zeroAdvertisingID,
			state: telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_LAT,
			lmt:   true,
		},
		{
			name: "real id",
			base: &schema.BaseRequest{
				User: schema.User{IDFA: realID, TrackingAuthorizationStatus: "authorized"},
			},
			want:  realID,
			state: telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_REAL,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, state, lmt := advertisingID(tt.base)
			if got != tt.want || state != tt.state || lmt != tt.lmt {
				t.Errorf("advertisingID() = %q %s lmt=%v, want %q %s lmt=%v", got, state, lmt, tt.want, tt.state, tt.lmt)
			}
		})
	}
}

func TestFillEnvelopeFromRequest(t *testing.T) {
	base := &schema.BaseRequest{
		Device: schema.Device{
			Manufacturer:   "Google",
			Model:          "Pixel",
			OS:             "android",
			OSVersion:      "14",
			Width:          1080,
			Height:         2400,
			PXRatio:        2.75,
			ConnectionType: "CELLULAR_5_G",
			Type:           device.PhoneType,
		},
		App: schema.App{Bundle: "org.example", Key: "app-key", Version: "3.2", SDKVersion: "0.9"},
		User: schema.User{
			IDFA:                        "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			TrackingAuthorizationStatus: "authorized",
		},
		Session:     schema.Session{StartTS: 1688371931187},
		Regulations: &schema.Regulations{GDPR: true, USPrivacy: "1YNN", EUPrivacy: "CPtcf", COPPA: false},
	}
	base.Ext = `{"mediation_mode":"max"}`
	base.NormalizeValues()

	env := &telemetryv1.Envelope{Country: "US"}
	fillEnvelopeFromRequest(env, base)

	if env.GetCountry() != "US" {
		t.Errorf("country overwritten: %q", env.GetCountry())
	}
	if got := env.GetIntegration(); got.GetHost() != telemetryv1.Host_HOST_MAX_ADAPTER ||
		got.GetAuctionOwner() != telemetryv1.AuctionOwner_AUCTION_OWNER_BIDON ||
		got.GetIntegrationMode() != telemetryv1.IntegrationMode_INTEGRATION_MODE_MAX_CUSTOM {
		t.Errorf("integration = %v", got)
	}
	if got := env.GetApp(); got.GetBundle() != "org.example" || got.GetAppKey() != "app-key" || got.GetSdkVersion() != "0.9" || got.GetAppVersion() != "3.2" {
		t.Errorf("app = %v", got)
	}
	d := env.GetDevice()
	if d.GetOs() != "android" || d.GetManufacturer() != "Google" || d.GetModel() != "Pixel" || d.GetOsVersion() != "14" {
		t.Errorf("device = %v", d)
	}
	if d.GetConnectionType() != telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR_5G {
		t.Errorf("connection = %s", d.GetConnectionType())
	}
	if d.GetScreenWidth() != 1080 || d.GetScreenHeight() != 2400 || d.GetScreenDensity() != 2.75 {
		t.Errorf("screen: %d x %d @ %v", d.GetScreenWidth(), d.GetScreenHeight(), d.GetScreenDensity())
	}
	if d.GetDeviceType() != 4 {
		t.Errorf("device_type = %d, want 4", d.GetDeviceType())
	}
	if env.GetSession().GetStartTs() != 1688371931187 {
		t.Errorf("session.start_ts = %d", env.GetSession().GetStartTs())
	}
	if env.GetSession().GetSdkUptimeMs() != 0 {
		t.Errorf("session.sdk_uptime_ms = %d, want unset", env.GetSession().GetSdkUptimeMs())
	}
	p := env.GetPrivacy()
	if !p.GetGdprApplies() || p.GetUsPrivacy() != "1YNN" || p.GetCoppa() || !p.GetHasTcfString() {
		t.Errorf("consent: gdpr=%v us=%q coppa=%v tcf=%v", p.GetGdprApplies(), p.GetUsPrivacy(), p.GetCoppa(), p.GetHasTcfString())
	}
	if p.GetAdvertisingId() != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" || p.GetLmt() {
		t.Errorf("advertising_id = %q lmt=%v", p.GetAdvertisingId(), p.GetLmt())
	}
	if env.GetDemand() != nil {
		t.Errorf("demand = %v, want unset", env.GetDemand())
	}
}

func TestFillEnvelopeWithoutRequest(t *testing.T) {
	env := &telemetryv1.Envelope{}
	fillEnvelopeFromRequest(env, nil)
	if env.GetPrivacy().GetAdvertisingId() != zeroAdvertisingID {
		t.Errorf("advertising_id = %q, want zero UUID", env.GetPrivacy().GetAdvertisingId())
	}
	if env.GetApp() != nil || env.GetDevice() != nil || env.GetIntegration() != nil {
		t.Errorf("groups must stay unset without a request: %v", env)
	}
}

func TestCOPPAFlagged(t *testing.T) {
	yes, no := true, false
	tests := []struct {
		name string
		base *schema.BaseRequest
		want bool
	}{
		{name: "none", base: &schema.BaseRequest{}},
		{name: "regs only", base: &schema.BaseRequest{Regulations: &schema.Regulations{COPPA: true}}, want: true},
		{name: "user only, no regs", base: &schema.BaseRequest{User: schema.User{COPPA: &yes}}, want: true},
		{name: "user false", base: &schema.BaseRequest{User: schema.User{COPPA: &no}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := coppaFlagged(tt.base); got != tt.want {
				t.Errorf("coppaFlagged() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnectionType(t *testing.T) {
	tests := map[string]telemetryv1.ConnectionType{
		"":                 telemetryv1.ConnectionType_CONNECTION_TYPE_UNSPECIFIED,
		"WIFI":             telemetryv1.ConnectionType_CONNECTION_TYPE_WIFI,
		"ETHERNET":         telemetryv1.ConnectionType_CONNECTION_TYPE_ETHERNET,
		"CELLULAR":         telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR,
		"CELLULAR_UNKNOWN": telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR,
		"CELLULAR_2_G":     telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR_2G,
		"CELLULAR_3_G":     telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR_3G,
		"CELLULAR_4_G":     telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR_4G,
		"CELLULAR_5_G":     telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR_5G,
		"SATELLITE":        telemetryv1.ConnectionType_CONNECTION_TYPE_UNKNOWN,
	}
	for in, want := range tests {
		if got := connectionType(in); got != want {
			t.Errorf("connectionType(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestIntegrationMode(t *testing.T) {
	tests := []struct {
		name     string
		ext      string
		host     telemetryv1.Host
		mode     telemetryv1.IntegrationMode
		mediator telemetryv1.Mediator
	}{
		{name: "direct", host: telemetryv1.Host_HOST_PUBLISHER, mode: telemetryv1.IntegrationMode_INTEGRATION_MODE_DIRECT, mediator: telemetryv1.Mediator_MEDIATOR_BIDON},
		{name: "max", ext: `{"mediation_mode":"max"}`, host: telemetryv1.Host_HOST_MAX_ADAPTER, mode: telemetryv1.IntegrationMode_INTEGRATION_MODE_MAX_CUSTOM, mediator: telemetryv1.Mediator_MEDIATOR_MAX},
		{name: "mediator max only", ext: `{"mediator":"max"}`, host: telemetryv1.Host_HOST_MAX_ADAPTER, mode: telemetryv1.IntegrationMode_INTEGRATION_MODE_MAX_CUSTOM, mediator: telemetryv1.Mediator_MEDIATOR_MAX},
		{name: "max bidding", ext: `{"mediation_mode":"max_bidding"}`, host: telemetryv1.Host_HOST_MAX_ADAPTER, mode: telemetryv1.IntegrationMode_INTEGRATION_MODE_MAX_BIDDING, mediator: telemetryv1.Mediator_MEDIATOR_MAX},
		{name: "levelplay", ext: `{"mediator":"levelplay"}`, host: telemetryv1.Host_HOST_PUBLISHER, mode: telemetryv1.IntegrationMode_INTEGRATION_MODE_DIRECT, mediator: telemetryv1.Mediator_MEDIATOR_LEVEL_PLAY},
		{name: "unknown mediator", ext: `{"mediator":"other"}`, host: telemetryv1.Host_HOST_PUBLISHER, mode: telemetryv1.IntegrationMode_INTEGRATION_MODE_DIRECT, mediator: telemetryv1.Mediator_MEDIATOR_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := &schema.BaseRequest{Ext: tt.ext}
			base.NormalizeValues()
			env := &telemetryv1.Envelope{}
			fillEnvelopeFromRequest(env, base)
			got := env.GetIntegration()
			if got.GetAuctionOwner() != telemetryv1.AuctionOwner_AUCTION_OWNER_BIDON {
				t.Errorf("owner = %s", got.GetAuctionOwner())
			}
			if got.GetHost() != tt.host || got.GetIntegrationMode() != tt.mode || got.GetMediator() != tt.mediator {
				t.Errorf("got %s %s %s", got.GetHost(), got.GetIntegrationMode(), got.GetMediator())
			}
		})
	}
}
