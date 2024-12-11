package grpcserver

import (
	adcom "github.com/bidon-io/bidon-backend/pkg/proto/com/iabtechlab/adcom/v1"
	adcomctx "github.com/bidon-io/bidon-backend/pkg/proto/com/iabtechlab/adcom/v1/context"
	pbctx "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/proto/v1/context"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	v3 "github.com/bidon-io/bidon-backend/pkg/proto/com/iabtechlab/openrtb/v3"

	"github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/proto/v1/mediation"
)

func TestAuctionRequestAdapter_OpenRTBToAuctionRequest(t *testing.T) {
	adapter := NewAuctionRequestAdapter()

	buildValidRequest := func() *v3.Openrtb {
		app := &adcomctx.DistributionChannel_App{
			Bundle: proto.String("com.example.app"),
			Ver:    proto.String("1.0"),
		}
		appExt := &mediation.AppExt{
			Key:              proto.String("app_key"),
			Framework:        proto.String("flutter"),
			FrameworkVersion: proto.String("1.22"),
			PluginVersion:    proto.String("1.0.0"),
			SdkVersion:       proto.String("2.0.0"),
			Skadn:            []string{"skadn1", "skadn2"},
		}
		proto.SetExtension(app, mediation.E_AppExt, appExt)

		user := &adcomctx.User{}
		userExt := &mediation.UserExt{
			Idfa:                        proto.String("IDFA-12345"),
			Idfv:                        proto.String("IDFV-12345"),
			Idg:                         proto.String("IDG-12345"),
			TrackingAuthorizationStatus: proto.String("authorized"),
			Segments: []*mediation.Segment{
				{
					Id:  proto.String("segment_id"),
					Uid: proto.String("segment_uid"),
				},
			},
		}
		proto.SetExtension(user, mediation.E_UserExt, userExt)

		regs := &adcomctx.Regs{
			Coppa: proto.Bool(true),
			Gdpr:  proto.Bool(true),
		}
		regsExt := &mediation.RegsExt{
			UsPrivacy: proto.String("1YNN"),
			EuPrivacy: proto.String("1"),
			Iab:       proto.String(`{"key":"value"}`),
		}
		proto.SetExtension(regs, mediation.E_RegsExt, regsExt)

		device := &adcomctx.Device{
			Ua:    proto.String("Mozilla/5.0"),
			Make:  proto.String("Apple"),
			Model: proto.String("iPhone"),
			Os:    proto.Int32(int32(adcom.OperatingSystem_IOS)),
			Osv:   proto.String("14.4"),
		}
		deviceExt := &mediation.DeviceExt{
			Id:          proto.String("session_id"),
			LaunchTs:    proto.Int32(1617187200),
			RamUsed:     proto.Int32(1024),
			RamSize:     proto.Int32(2048),
			StorageFree: proto.Int32(512),
			StorageUsed: proto.Int32(256),
			Battery:     proto.Float64(80.5),
			CpuUsage:    proto.Float64(10.6),
		}
		proto.SetExtension(device, mediation.E_DeviceExt, deviceExt)

		c := &pbctx.Context{
			DistributionChannel: &adcomctx.DistributionChannel{
				ChannelOneof: &adcomctx.DistributionChannel_App_{
					App: app,
				},
			},
			Device: device,
			User:   user,
			Regs:   regs,
		}
		ctxBytes, _ := proto.Marshal(c)

		placement := &adcom.Placement{}
		placementExt := &mediation.PlacementExt{
			AuctionId:               proto.String("auction_id_123"),
			AuctionKey:              proto.String("auction_key_789"),
			AuctionConfigurationUid: proto.String("config_uid_456"),
			Orientation:             ptr(mediation.Orientation_PORTRAIT),
			Demands: map[string]*mediation.Demand{
				"demand_key": {
					Token:         proto.String("token_value"),
					Status:        proto.String("status_value"),
					TokenFinishTs: proto.Int64(1234567890),
					TokenStartTs:  proto.Int64(1234567000),
				},
			},
		}
		proto.SetExtension(placement, mediation.E_PlacementExt, placementExt)

		placementBytes, err := proto.Marshal(placement)
		if err != nil {
			t.Fatalf("failed to marshal placement: %v", err)
		}

		item := &v3.Item{
			Id:   proto.String("auction_id_123"),
			Flr:  proto.Float32(0.5),
			Spec: placementBytes,
		}

		return &v3.Openrtb{
			PayloadOneof: &v3.Openrtb_Request{
				Request: &v3.Request{
					Test:    proto.Bool(true),
					Tmax:    proto.Uint32(1000),
					Context: ctxBytes,
					Item:    []*v3.Item{item},
				},
			},
		}
	}

	tests := []struct {
		name        string
		input       *v3.Openrtb
		expectError bool
		verify      func(t *testing.T, ar *schema.AuctionV2Request, err error)
	}{
		{
			name:  "valid request with extensions",
			input: buildValidRequest(),
			verify: func(t *testing.T, ar *schema.AuctionV2Request, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if ar == nil {
					t.Fatal("expected non-nil AuctionV2Request, got nil")
				}
				if ar.TMax != 1000 {
					t.Errorf("expected TMax=1000, got %d", ar.TMax)
				}
				if ar.Test != true {
					t.Errorf("expected Test=true, got %v", ar.Test)
				}
				if ar.App.Bundle != "com.example.app" {
					t.Errorf("expected App.Bundle=com.example.app, got %s", ar.App.Bundle)
				}
				if ar.User.IDFA != "IDFA-12345" {
					t.Errorf("expected User.IDFA=IDFA-12345, got %s", ar.User.IDFA)
				}
				if ar.AdObject.AuctionID != "auction_id_123" {
					t.Errorf("expected AdObject.AuctionID=auction_id_123, got %s", ar.AdObject.AuctionID)
				}
				if ar.AdObject.AuctionKey != "auction_key_789" {
					t.Errorf("expected AdObject.AuctionKey=auction_key_789, got %s", ar.AdObject.AuctionKey)
				}
				if ar.AdObject.PriceFloor != 0.5 {
					t.Errorf("expected AdObject.PriceFloor=0.5, got %f", ar.AdObject.PriceFloor)
				}
				demand, ok := ar.AdObject.Demands["demand_key"]
				if !ok {
					t.Errorf("expected demand_key in AdObject.Demands, not found")
				} else {
					if demand["token"] != "token_value" {
						t.Errorf("expected token_value in demands, got %v", demand["token"])
					}
				}
			},
		},
		{
			name:  "nil request",
			input: &v3.Openrtb{
				// no Request field
			},
			expectError: true,
			verify: func(t *testing.T, ar *schema.AuctionV2Request, err error) {
				if err == nil {
					t.Fatal("expected an error, got none")
				}
				if ar != nil {
					t.Fatal("expected ar=nil, got a non-nil value")
				}
				if msg := err.Error(); msg == "" || !strings.Contains(msg, "request is nil") {
					t.Errorf("expected error containing 'request is nil', got %q", msg)
				}
			},
		},
		{
			name: "empty context",
			input: &v3.Openrtb{
				PayloadOneof: &v3.Openrtb_Request{
					Request: &v3.Request{
						Item: []*v3.Item{{Id: proto.String("some_id")}},
					},
				},
			},
			expectError: true,
			verify: func(t *testing.T, ar *schema.AuctionV2Request, err error) {
				if err == nil {
					t.Fatal("expected an error, got none")
				}
				if ar != nil {
					t.Fatal("expected ar=nil, got non-nil")
				}
				if msg := err.Error(); !strings.Contains(msg, "request context is empty") {
					t.Errorf("expected error containing 'request context is empty', got %q", msg)
				}
			},
		},
		{
			name: "no items",
			input: func() *v3.Openrtb {
				r := buildValidRequest()
				r.GetRequest().Item = nil
				return r
			}(),
			expectError: true,
			verify: func(t *testing.T, ar *schema.AuctionV2Request, err error) {
				if err == nil {
					t.Fatal("expected an error, got none")
				}
				if ar != nil {
					t.Fatal("expected ar=nil, got non-nil")
				}
				if msg := err.Error(); !strings.Contains(msg, "no items in request") {
					t.Errorf("expected error containing 'no items in request', got %q", msg)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ar, err := adapter.OpenRTBToAuctionRequest(tc.input)
			if tc.expectError && err == nil {
				t.Fatal("expected an error but got none")
			} else if !tc.expectError && err != nil {
				t.Fatalf("expected no error but got %v", err)
			}

			tc.verify(t, ar, err)
		})
	}
}

func ptr[T any](t T) *T {
	return &t
}
