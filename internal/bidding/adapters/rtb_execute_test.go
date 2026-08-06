package adapters_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
)

func TestCountryFromRequest(t *testing.T) {
	t.Parallel()

	if got := adapters.CountryFromRequest(openrtb.BidRequest{}); got != "" {
		t.Fatalf("CountryFromRequest(empty) = %q, want empty", got)
	}

	req := openrtb.BidRequest{
		Device: &openrtb2.Device{
			Geo: &openrtb2.Geo{Country: "USA"},
		},
	}
	if got := adapters.CountryFromRequest(req); got != "USA" {
		t.Fatalf("CountryFromRequest = %q, want USA", got)
	}
}

func TestExecuteRTBRequest_HappyPath(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		if got := r.Header.Get("X-OpenRTB-Version"); got != "2.5" {
			t.Errorf("X-OpenRTB-Version = %q, want 2.5", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(string(body), `"id":"req-1"`) {
			t.Errorf("body missing request id: %s", body)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"resp-1"}`))
	}))
	t.Cleanup(server.Close)

	headers := make(http.Header)
	headers.Set("X-OpenRTB-Version", "2.5")

	dr := adapters.ExecuteRTBRequest(context.Background(), server.Client(), openrtb.BidRequest{ID: "req-1"}, adapters.ExecuteRTBOptions{
		DemandID:    adapter.VungleKey,
		URL:         server.URL,
		TagID:       "tag-1",
		PlacementID: "placement-1",
		Headers:     headers,
	})

	if dr.Error != nil {
		t.Fatalf("Error = %v, want nil", dr.Error)
	}
	if dr.DemandID != adapter.VungleKey {
		t.Fatalf("DemandID = %q, want %q", dr.DemandID, adapter.VungleKey)
	}
	if dr.RequestID != "req-1" {
		t.Fatalf("RequestID = %q, want req-1", dr.RequestID)
	}
	if dr.TagID != "tag-1" {
		t.Fatalf("TagID = %q, want tag-1", dr.TagID)
	}
	if dr.PlacementID != "placement-1" {
		t.Fatalf("PlacementID = %q, want placement-1", dr.PlacementID)
	}
	if dr.Status != http.StatusOK {
		t.Fatalf("Status = %d, want 200", dr.Status)
	}
	if dr.RawResponse != `{"id":"resp-1"}` {
		t.Fatalf("RawResponse = %q, want {\"id\":\"resp-1\"}", dr.RawResponse)
	}
	if !strings.Contains(dr.RawRequest, `"id":"req-1"`) {
		t.Fatalf("RawRequest missing id: %s", dr.RawRequest)
	}
}

func TestExecuteRTBRequest_EmptyURL(t *testing.T) {
	t.Parallel()

	dr := adapters.ExecuteRTBRequest(context.Background(), http.DefaultClient, openrtb.BidRequest{ID: "req-1"}, adapters.ExecuteRTBOptions{
		DemandID: adapter.MolocoKey,
	})
	if dr.Error == nil || dr.Error.Error() != "endpoint URL is empty" {
		t.Fatalf("Error = %v, want endpoint URL is empty", dr.Error)
	}
	if dr.RawRequest == "" {
		t.Fatal("expected RawRequest to be set before URL check")
	}
}

func TestExecuteRTBRequest_PrepareURL(t *testing.T) {
	t.Parallel()

	var gotURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	dr := adapters.ExecuteRTBRequest(context.Background(), server.Client(), openrtb.BidRequest{ID: "req-1", Test: 1}, adapters.ExecuteRTBOptions{
		DemandID: adapter.StartIOKey,
		URL:      server.URL,
		PrepareURL: func(base string, request openrtb.BidRequest) (string, error) {
			if request.Test != 1 {
				t.Fatalf("PrepareURL Test = %d, want 1", request.Test)
			}
			return base + "?account=acc-1&testAdsEnabled=true", nil
		},
	})
	if dr.Error != nil {
		t.Fatalf("Error = %v, want nil", dr.Error)
	}
	if !strings.Contains(gotURL, "account=acc-1") || !strings.Contains(gotURL, "testAdsEnabled=true") {
		t.Fatalf("gotURL = %q, want account and testAdsEnabled query", gotURL)
	}
}

func TestExecuteRTBRequest_PrepareURLError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("bad url")
	dr := adapters.ExecuteRTBRequest(context.Background(), http.DefaultClient, openrtb.BidRequest{ID: "req-1"}, adapters.ExecuteRTBOptions{
		DemandID: adapter.StartIOKey,
		URL:      "https://example.com",
		PrepareURL: func(string, openrtb.BidRequest) (string, error) {
			return "", wantErr
		},
	})
	if !errors.Is(dr.Error, wantErr) {
		t.Fatalf("Error = %v, want %v", dr.Error, wantErr)
	}
}

func TestExecuteRTBRequest_AfterDo(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Fb-An-Errors", "bad placement")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":1}`))
	}))
	t.Cleanup(server.Close)

	dr := adapters.ExecuteRTBRequest(context.Background(), server.Client(), openrtb.BidRequest{ID: "req-1"}, adapters.ExecuteRTBOptions{
		DemandID:   adapter.MetaKey,
		URL:        server.URL,
		TimeoutURL: "https://example.com/timeout",
		AfterDo: func(resp *http.Response, dr *adapters.DemandResponse) {
			if dr.Status == http.StatusBadRequest {
				dr.Error = errors.New(resp.Header.Get("X-Fb-An-Errors"))
			}
		},
	})

	if dr.TimeoutURL != "https://example.com/timeout" {
		t.Fatalf("TimeoutURL = %q, want timeout URL", dr.TimeoutURL)
	}
	if dr.Status != http.StatusBadRequest {
		t.Fatalf("Status = %d, want 400", dr.Status)
	}
	if dr.Error == nil || dr.Error.Error() != "bad placement" {
		t.Fatalf("Error = %v, want bad placement", dr.Error)
	}
	if dr.RawResponse != `{"error":1}` {
		t.Fatalf("RawResponse = %q", dr.RawResponse)
	}
}

func TestExecuteRTBRequest_DoError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()

	dr := adapters.ExecuteRTBRequest(context.Background(), server.Client(), openrtb.BidRequest{ID: "req-1"}, adapters.ExecuteRTBOptions{
		DemandID: adapter.VKAdsKey,
		URL:      server.URL,
	})
	if dr.Error == nil {
		t.Fatal("expected Do error")
	}
}
