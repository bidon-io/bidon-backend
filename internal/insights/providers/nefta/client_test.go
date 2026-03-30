package nefta

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prebid/openrtb/v19/openrtb2"
)

func TestClientInitSuccess(t *testing.T) {
	var captured InitRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected application/json content type")
		}
		if r.Header.Get("nefta-sdk-version") != "0.8.0" {
			t.Fatalf("expected nefta-sdk-version 0.8.0, got %q", r.Header.Get("nefta-sdk-version"))
		}
		if r.Header.Get("nefta-sdk-platform") != "ios" {
			t.Fatalf("expected nefta-sdk-platform ios, got %q", r.Header.Get("nefta-sdk-platform"))
		}
		if r.Header.Get("nefta-sdk-bundle") != "com.example.app" {
			t.Fatalf("expected nefta-sdk-bundle com.example.app, got %q", r.Header.Get("nefta-sdk-bundle"))
		}
		if r.Header.Get("nefta-sdk-app-version") != "1.0.0" {
			t.Fatalf("expected nefta-sdk-app-version 1.0.0, got %q", r.Header.Get("nefta-sdk-app-version"))
		}
		if r.Header.Get("nefta-sdk-nuid") != "old-nuid" {
			t.Fatalf("expected nefta-sdk-nuid old-nuid, got %q", r.Header.Get("nefta-sdk-nuid"))
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"nuid":"new-nuid"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.InitURL = server.URL

	resp, err := client.Init(context.Background(), InitRequest{
		NUID:        "old-nuid",
		SessionID:   42,
		AppBundle:   "com.example.app",
		AppPlatform: "ios",
		AppVersion:  "1.0.0",
		SDKVersion:  "0.8.0",
		Device:      &openrtb2.Device{OS: "iOS"},
		UserGeo:     &openrtb2.Geo{Country: "USA"},
	})
	if err != nil {
		t.Fatalf("client init failed: %v", err)
	}

	if resp.Response.NUID != "new-nuid" {
		t.Fatalf("expected nuid new-nuid, got %s", resp.Response.NUID)
	}
	if !strings.Contains(resp.RawRequestHeaders, `"Nefta-Sdk-Version":["0.8.0"]`) {
		t.Fatalf("expected request headers to include nefta-sdk-version, got %q", resp.RawRequestHeaders)
	}
	if captured.SessionID != 42 {
		t.Fatalf("expected session_id 42, got %d", captured.SessionID)
	}
	if captured.AppBundle != "com.example.app" {
		t.Fatalf("expected app_bundle com.example.app, got %s", captured.AppBundle)
	}
	if captured.AppPlatform != "ios" {
		t.Fatalf("expected app_platform ios, got %q", captured.AppPlatform)
	}
	if captured.Device == nil || captured.Device.OS != "iOS" {
		t.Fatalf("expected device.os iOS, got %+v", captured.Device)
	}
	if captured.UserGeo == nil || captured.UserGeo.Country != "USA" {
		t.Fatalf("expected user_geo.country USA, got %+v", captured.UserGeo)
	}
}

func TestClientInitReturnsErrorOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`upstream down`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.InitURL = server.URL

	_, err := client.Init(context.Background(), InitRequest{})
	if err == nil {
		t.Fatalf("expected error for non-2xx response")
	}
}

func TestClientInitReturnsErrorOnInvalidNUID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"nuid":"  "}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.InitURL = server.URL

	_, err := client.Init(context.Background(), InitRequest{})
	if !errors.Is(err, ErrInvalidNUID) {
		t.Fatalf("expected ErrInvalidNUID, got %v", err)
	}
}

func TestClientInitHonorsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"nuid":"new-nuid"}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.InitURL = server.URL
	client.Timeout = 10 * time.Millisecond

	_, err := client.Init(context.Background(), InitRequest{})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
}

func TestClientFloorPriceSuccess(t *testing.T) {
	var captured FloorPriceRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected application/json content type")
		}
		if r.Header.Get("nefta-sdk-version") != "0.8.0" {
			t.Fatalf("expected nefta-sdk-version 0.8.0, got %q", r.Header.Get("nefta-sdk-version"))
		}
		if r.Header.Get("nefta-sdk-platform") != "android" {
			t.Fatalf("expected nefta-sdk-platform android, got %q", r.Header.Get("nefta-sdk-platform"))
		}
		if r.Header.Get("nefta-sdk-bundle") != "com.example.app" {
			t.Fatalf("expected nefta-sdk-bundle com.example.app, got %q", r.Header.Get("nefta-sdk-bundle"))
		}
		if r.Header.Get("nefta-sdk-app-version") != "2.0.0" {
			t.Fatalf("expected nefta-sdk-app-version 2.0.0, got %q", r.Header.Get("nefta-sdk-app-version"))
		}
		if r.Header.Get("nefta-sdk-nuid") != "nuid-1" {
			t.Fatalf("expected nefta-sdk-nuid nuid-1, got %q", r.Header.Get("nefta-sdk-nuid"))
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"floor_prices":[{"auction_id":1,"floor_price":0.12,"notification":{"auction":"a","impression":"i","click":"c"}}],"control":true}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.FloorPriceURL = server.URL

	resp, err := client.FloorPrice(context.Background(), FloorPriceRequest{
		NUID:            "nuid-1",
		SessionID:       4,
		AppPlatform:     "android",
		SDKVersion:      "0.8.0",
		AdOpportunityID: 2,
		AdType:          "rewarded",
		SessionStartTS:  12345,
		App:             &openrtb2.App{Bundle: "com.example.app", Ver: "2.0.0"},
		UserGeo:         &openrtb2.Geo{Country: "USA"},
		Auctions: []FloorPriceAuction{
			{ID: 1, FloorPrice: 0.05, Bidders: []string{"bidmachine", "meta"}},
		},
	})
	if err != nil {
		t.Fatalf("client floor-price failed: %v", err)
	}
	if len(resp.Response.FloorPrices) != 1 || resp.Response.FloorPrices[0].FloorPrice != 0.12 {
		t.Fatalf("unexpected floor-price response: %+v", resp.Response)
	}
	if captured.AdOpportunityID != 2 {
		t.Fatalf("expected ad_opportunity_id 2, got %d", captured.AdOpportunityID)
	}
	if len(captured.Auctions) != 1 || captured.Auctions[0].Bidders[0] != "bidmachine" {
		t.Fatalf("unexpected captured auctions: %+v", captured.Auctions)
	}
}
