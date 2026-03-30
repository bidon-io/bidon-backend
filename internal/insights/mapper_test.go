package insights

import (
	"testing"

	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/device"
	insightsopenrtb "github.com/bidon-io/bidon-backend/internal/insights/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

func TestOpenRTBFromBaseRequest(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		got := OpenRTBFromBaseRequest(nil)
		if got.App != nil || got.Device != nil || got.UserGeo != nil {
			t.Fatalf("expected empty payload for nil request, got %+v", got)
		}
	})

	t.Run("maps app and device only (geo via geocoder enrichment)", func(t *testing.T) {
		js := 1
		req := &schema.BaseRequest{
			App: schema.App{
				Bundle:     "com.example.app",
				Version:    "1.2.3",
				SDKVersion: "0.8.0",
			},
			Device: schema.Device{
				UserAgent:       "ua",
				Manufacturer:    "Apple",
				Model:           "iPhone",
				OS:              "iOS",
				OSVersion:       "17.0",
				HardwareVersion: "A17",
				Height:          2436,
				Width:           1125,
				PPI:             460,
				PXRatio:         3,
				JS:              &js,
				Language:        "en",
				IP:              "127.0.0.1",
				Carrier:         "carrier",
				MCCMNC:          "310260",
				Type:            device.PhoneType,
			},
			User: schema.User{
				IDFA: "6f9619ff-8b86-d011-b42d-00cf4fc964ff",
			},
		}

		got := OpenRTBFromBaseRequest(req)

		if got.App == nil {
			t.Fatalf("expected app to be mapped")
		}
		if got.App.Bundle != req.App.Bundle {
			t.Fatalf("expected bundle %q, got %q", req.App.Bundle, got.App.Bundle)
		}
		if got.App.Ver != req.App.Version {
			t.Fatalf("expected app version %q, got %q", req.App.Version, got.App.Ver)
		}
		if got.App.Domain != req.App.Bundle {
			t.Fatalf("expected app domain %q, got %q", req.App.Bundle, got.App.Domain)
		}

		if got.Device == nil {
			t.Fatalf("expected device to be mapped")
		}
		if got.Device.IFA != req.User.IDFA {
			t.Fatalf("expected IFA %q, got %q", req.User.IDFA, got.Device.IFA)
		}
		if got.Device.OS != req.Device.OS {
			t.Fatalf("expected OS %q, got %q", req.Device.OS, got.Device.OS)
		}
		if got.Device.JS != int8(js) {
			t.Fatalf("expected JS %d, got %d", js, got.Device.JS)
		}

		if got.UserGeo != nil {
			t.Fatalf("expected user_geo to be nil before geocoder enrichment, got %+v", got.UserGeo)
		}
	})

	t.Run("returns nil user_geo when geo is empty", func(t *testing.T) {
		req := &schema.BaseRequest{
			App: schema.App{
				Bundle:  "com.example.app",
				Version: "1.2.3",
			},
			Device: schema.Device{
				Manufacturer: "Google",
				Model:        "Pixel",
				OS:           "android",
				OSVersion:    "14",
			},
		}

		got := OpenRTBFromBaseRequest(req)
		if got.UserGeo != nil {
			t.Fatalf("expected nil user_geo for empty geo, got %+v", got.UserGeo)
		}
	})
}

func TestInitRequestFromBaseRequest(t *testing.T) {
	req := &schema.BaseRequest{
		App: schema.App{
			Bundle:     "com.example.app",
			Version:    "1.2.3",
			SDKVersion: "0.8.0",
		},
		Device: schema.Device{
			Manufacturer: "Apple",
			Model:        "iPhone",
			OS:           "iOS",
			OSVersion:    "17.0",
		},
		User: schema.User{
			IDFA: "idfa",
			IDG:  "idg",
			IDFV: "idfv",
		},
	}

	got := InitRequestFromBaseRequest(42, req)
	if got.AppID != 42 {
		t.Fatalf("expected app id 42, got %d", got.AppID)
	}
	if got.AppVersion != "1.2.3" || got.SDKVersion != "0.8.0" {
		t.Fatalf("unexpected app/sdk version mapping: %+v", got)
	}
	if got.IDFA != "idfa" || got.IDG != "idg" || got.IDFV != "idfv" {
		t.Fatalf("unexpected advertising ids mapping: %+v", got)
	}
	if got.OpenRTB.App == nil || got.OpenRTB.App.Bundle != "com.example.app" {
		t.Fatalf("expected openrtb app bundle mapping, got %+v", got.OpenRTB.App)
	}
}

func TestInitRequestFromAuctionRequest(t *testing.T) {
	req := &schema.AuctionRequest{
		BaseRequest: schema.BaseRequest{
			App: schema.App{
				Bundle:     "com.example.app",
				Version:    "1.2.3",
				SDKVersion: "0.8.0",
			},
			User: schema.User{
				IDFA: "idfa-auction",
			},
		},
	}

	got := InitRequestFromAuctionRequest(777, req)
	if got.AppID != 777 {
		t.Fatalf("expected app id 777, got %d", got.AppID)
	}
	if got.IDFA != "idfa-auction" {
		t.Fatalf("expected idfa-auction, got %q", got.IDFA)
	}
}

func TestEnrichOpenRTBWithGeoData(t *testing.T) {
	req := insightsopenrtb.InitRequest{
		UserGeo: &openrtb2.Geo{
			Country: "USA",
		},
	}

	enrichOpenRTBWithGeoData(&req, geocoder.GeoData{
		CountryCode3: "CAN",
		RegionCode:   "CA",
		CityName:     "San Francisco",
		ZipCode:      "94103",
		Lat:          37.77,
		Lon:          -122.41,
		Accuracy:     25,
	})

	if req.UserGeo == nil {
		t.Fatalf("expected user geo to be present")
	}
	if req.UserGeo.Region != "CA" {
		t.Fatalf("expected region CA, got %q", req.UserGeo.Region)
	}
	if req.UserGeo.Country != "CAN" {
		t.Fatalf("expected country from geodata, got %q", req.UserGeo.Country)
	}
	if req.UserGeo.City != "San Francisco" {
		t.Fatalf("expected city fallback, got %q", req.UserGeo.City)
	}
	if req.UserGeo.ZIP != "94103" {
		t.Fatalf("expected zip fallback, got %q", req.UserGeo.ZIP)
	}
	if req.UserGeo.Type != adcom1.LocationIP {
		t.Fatalf("expected geo type LocationIP, got %v", req.UserGeo.Type)
	}
	if req.UserGeo.IPService != adcom1.LocationServiceMaxMind {
		t.Fatalf("expected ip service MaxMind, got %v", req.UserGeo.IPService)
	}

	enrichOpenRTBWithGeoData(&req, geocoder.GeoData{})
	if req.UserGeo != nil {
		t.Fatalf("expected nil user geo for empty geodata, got %+v", req.UserGeo)
	}
}
