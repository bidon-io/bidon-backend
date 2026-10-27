package telemetry

import (
	"testing"

	"github.com/bidon-io/bidon-backend/internal/device"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

func TestAdvertisingIDFollowsPRDOrder(t *testing.T) {
	real := "11111111-1111-1111-1111-111111111111"
	coppa := true
	tests := []struct {
		name  string
		base  *schema.BaseRequest
		want  string
		state telemetryv1.AdvertisingIdState
	}{
		{
			name:  "no request",
			want:  zeroAdvertisingID,
			state: telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_UNSPECIFIED,
		},
		{
			name: "coppa wins over a real id",
			base: &schema.BaseRequest{
				User: schema.User{IDFA: real, TrackingAuthorizationStatus: "authorized", COPPA: &coppa},
			},
			want:  zeroAdvertisingID,
			state: telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_COPPA,
		},
		{
			name: "missing consent",
			base: &schema.BaseRequest{
				User: schema.User{IDFA: real, TrackingAuthorizationStatus: "denied"},
			},
			want:  zeroAdvertisingID,
			state: telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_NO_CONSENT,
		},
		{
			name: "authorized with a deleted id",
			base: &schema.BaseRequest{
				User: schema.User{IDFA: zeroAdvertisingID, TrackingAuthorizationStatus: "AUTHORIZED"},
			},
			want:  zeroAdvertisingID,
			state: telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_ZEROED_LAT,
		},
		{
			name: "real id",
			base: &schema.BaseRequest{
				User: schema.User{IDFA: real, TrackingAuthorizationStatus: "authorized"},
			},
			want:  real,
			state: telemetryv1.AdvertisingIdState_ADVERTISING_ID_STATE_REAL,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, state := advertisingID(tt.base)
			if got != tt.want || state != tt.state {
				t.Errorf("advertisingID() = %q %s, want %q %s", got, state, tt.want, tt.state)
			}
		})
	}
}

func TestApplyPRDEnvelopeCopiesRequest(t *testing.T) {
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
		Regulations: &schema.Regulations{GDPR: true, USPrivacy: "1YNN", COPPA: false},
	}
	base.Ext = `{"mediation_mode":"max"}`
	base.NormalizeValues()

	env := &telemetryv1.Envelope{Country: "US"}
	applyPRDEnvelope(env, base)

	if env.GetCountry() != "US" {
		t.Errorf("country overwritten: %q", env.GetCountry())
	}
	if env.GetHost() != telemetryv1.Host_HOST_MAX_ADAPTER || env.GetAuctionOwner() != telemetryv1.AuctionOwner_AUCTION_OWNER_BIDON {
		t.Errorf("host/owner = %s/%s", env.GetHost(), env.GetAuctionOwner())
	}
	if env.GetIntegrationMode() != telemetryv1.IntegrationMode_INTEGRATION_MODE_MAX_CUSTOM {
		t.Errorf("integration_mode = %s", env.GetIntegrationMode())
	}
	if env.GetBundle() != "org.example" || env.GetSdkVersion() != "0.9" || env.GetAppVersion() != "3.2" {
		t.Errorf("app fields: bundle=%q sdk=%q version=%q", env.GetBundle(), env.GetSdkVersion(), env.GetAppVersion())
	}
	if env.GetOs() != "android" || env.GetDeviceManufacturer() != "Google" || env.GetDeviceModel() != "Pixel" {
		t.Errorf("device: %s %s %s", env.GetOs(), env.GetDeviceManufacturer(), env.GetDeviceModel())
	}
	if env.GetConnectionType() != telemetryv1.ConnectionType_CONNECTION_TYPE_CELLULAR_5G {
		t.Errorf("connection = %s", env.GetConnectionType())
	}
	if env.GetScreenWidth() != 1080 || env.GetScreenHeight() != 2400 || env.GetScreenDensity() != 2.75 {
		t.Errorf("screen: %d x %d @ %v", env.GetScreenWidth(), env.GetScreenHeight(), env.GetScreenDensity())
	}
	if env.GetDeviceType() != 4 {
		t.Errorf("device_type = %d, want 4", env.GetDeviceType())
	}
	if env.GetSessionStartTs() != 1688371931187 {
		t.Errorf("session_start_ts = %d", env.GetSessionStartTs())
	}
	if !env.GetGdprApplies() || env.GetUsPrivacy() != "1YNN" || env.GetCoppa() {
		t.Errorf("consent: gdpr=%v us=%q coppa=%v", env.GetGdprApplies(), env.GetUsPrivacy(), env.GetCoppa())
	}
	if env.GetAdvertisingId() != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("advertising_id = %q", env.GetAdvertisingId())
	}
	if env.GetSdkUptimeMs() != 0 {
		t.Errorf("sdk_uptime_ms = %d, want unset", env.GetSdkUptimeMs())
	}
}
