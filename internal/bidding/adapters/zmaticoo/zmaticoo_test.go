package zmaticoo

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/openrtb"
	"github.com/bidon-io/bidon-backend/internal/device"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

func TestZmaticooAdapter_CreateRequest(t *testing.T) {
	zmaticooAdapter := buildAdapter()

	auctionRequest := &schema.AuctionRequest{
		AdObject: schema.AdObject{
			AuctionID: "test-auction-id",
			Banner: &schema.BannerAdObject{
				Format: ad.BannerFormat,
			},
			Demands: map[adapter.Key]map[string]any{
				adapter.ZmaticooKey: {
					"token": `{"test-placement-id":{"token":"sdk-token","timestamp":1700000000000}}`,
				},
			},
		},
		BaseRequest: schema.BaseRequest{
			Device: schema.Device{
				Type: device.PhoneType,
			},
		},
		Adapters: schema.Adapters{
			adapter.ZmaticooKey: schema.Adapter{
				Version:    "1.0.0",
				SDKVersion: "1.0.0",
			},
		},
	}

	baseRequest := openrtb.BidRequest{
		ID:  "test-request-id",
		App: &openrtb2.App{},
	}

	request, err := zmaticooAdapter.CreateRequest(baseRequest, auctionRequest)
	if err != nil {
		t.Errorf("CreateRequest() error = %v", err)
		return
	}

	if len(request.Imp) != 1 {
		t.Errorf("Expected 1 impression, got %d", len(request.Imp))
	}

	imp := request.Imp[0]
	if imp.TagID != "test-placement-id" {
		t.Errorf("Expected TagID 'test-placement-id', got '%s'", imp.TagID)
	}

	if imp.BidFloorCur != "USD" {
		t.Errorf("Expected BidFloorCur 'USD', got '%s'", imp.BidFloorCur)
	}

	if len(request.Cur) != 1 || request.Cur[0] != "USD" {
		t.Errorf("Expected currency 'USD', got %v", request.Cur)
	}

	var reqExt map[string]any
	if err := json.Unmarshal(request.Ext, &reqExt); err != nil {
		t.Errorf("Failed to unmarshal request extension: %v", err)
		return
	}

	if token, ok := reqExt["sdk_token"].(string); !ok || token != "sdk-token" {
		t.Errorf("Expected sdk_token 'sdk-token' in request extension, got '%v'", reqExt["sdk_token"])
	}

	if timestamp, ok := reqExt["timestamp"].(float64); !ok || int64(timestamp) != 1700000000000 {
		t.Errorf("Expected timestamp 1700000000000 in request extension, got '%v'", reqExt["timestamp"])
	}

	if request.App == nil || request.App.ID != "test-app-id" {
		t.Errorf("Expected app.id 'test-app-id', got '%v'", request.App.ID)
	}
}

func TestZmaticooAdapter_ParseBids_Success(t *testing.T) {
	zmaticooAdapter := buildAdapter()

	dr := &adapters.DemandResponse{
		RequestID: "req-1",
		ImpID:     "imp-1",
		Status:    http.StatusOK,
		RawResponse: `{
			"code": 1,
			"msg": "bid success",
			"bid_result": {
				"request_id": "resp-1",
				"ecpm": 1.5,
				"nurl": "http://win.url",
				"adm": "<vast/>",
				"adid": "ad-1",
				"lurl": "http://loss.url",
				"burl": "http://bill.url"
			}
		}`,
	}

	result, err := zmaticooAdapter.ParseBids(dr)
	if err != nil {
		t.Errorf("ParseBids() error = %v", err)
		return
	}

	if result.Bid == nil {
		t.Error("Expected bid to be present")
		return
	}

	if result.Bid.Price != 1.5 {
		t.Errorf("Expected price 1.5, got %f", result.Bid.Price)
	}

	if result.Bid.Payload != "<vast/>" {
		t.Errorf("Expected payload '<vast/>', got '%s'", result.Bid.Payload)
	}

	if result.Bid.NURL != "http://win.url" {
		t.Errorf("Expected NURL 'http://win.url', got '%s'", result.Bid.NURL)
	}

	if result.Bid.LURL != "http://loss.url" {
		t.Errorf("Expected LURL 'http://loss.url', got '%s'", result.Bid.LURL)
	}

	if result.Bid.ImpID != "imp-1" {
		t.Errorf("Expected ImpID 'imp-1', got '%s'", result.Bid.ImpID)
	}

	if result.Bid.ID != "resp-1" {
		t.Errorf("Expected ID 'resp-1', got '%s'", result.Bid.ID)
	}
}

func TestZmaticooAdapter_ParseBids_NoContent(t *testing.T) {
	zmaticooAdapter := buildAdapter()

	dr := &adapters.DemandResponse{
		Status: http.StatusNoContent,
	}

	result, err := zmaticooAdapter.ParseBids(dr)
	if err != nil {
		t.Errorf("ParseBids() error = %v", err)
		return
	}

	if result.Bid != nil {
		t.Error("Expected no bid for no content response")
	}
}

func TestZmaticooAdapter_ParseBids_ErrorStatus(t *testing.T) {
	zmaticooAdapter := buildAdapter()

	dr := &adapters.DemandResponse{
		Status: http.StatusBadRequest,
	}

	_, err := zmaticooAdapter.ParseBids(dr)
	if err == nil {
		t.Error("Expected error for bad request status")
	}
}

func TestZmaticooAdapter_ParseBids_FailedCode(t *testing.T) {
	zmaticooAdapter := buildAdapter()

	dr := &adapters.DemandResponse{
		Status:      http.StatusOK,
		RawResponse: `{"code":1002,"msg":"Invalid token"}`,
	}

	_, err := zmaticooAdapter.ParseBids(dr)
	if err == nil {
		t.Error("Expected error for failed code response")
		return
	}

	if !strings.Contains(err.Error(), "zmaticoo bid failed") {
		t.Errorf("Expected error message to contain 'zmaticoo bid failed', got: %v", err)
	}
}

func TestZmaticooAdapter_ParseBids_MissingBidResult(t *testing.T) {
	zmaticooAdapter := buildAdapter()

	dr := &adapters.DemandResponse{
		Status:      http.StatusOK,
		RawResponse: `{"code":1,"msg":"bid success"}`,
	}

	_, err := zmaticooAdapter.ParseBids(dr)
	if err == nil {
		t.Error("Expected error for missing bid_result")
	}
}

func TestZmaticooAdapter_ExtractTokenAndTimestamp_Errors(t *testing.T) {
	zmaticooAdapter := buildAdapter()

	tests := []struct {
		name        string
		demands     map[adapter.Key]map[string]any
		errContains string
	}{
		{
			name:        "missing demand data",
			demands:     map[adapter.Key]map[string]any{},
			errContains: "missing token data",
		},
		{
			name: "empty token data",
			demands: map[adapter.Key]map[string]any{
				adapter.ZmaticooKey: {
					"token": "",
				},
			},
			errContains: "empty token data",
		},
		{
			name: "invalid token json",
			demands: map[adapter.Key]map[string]any{
				adapter.ZmaticooKey: {
					"token": "{invalid-json",
				},
			},
			errContains: "invalid token data",
		},
		{
			name: "missing placement",
			demands: map[adapter.Key]map[string]any{
				adapter.ZmaticooKey: {
					"token": `{"other-placement":{"token":"sdk-token","timestamp":1700000000000}}`,
				},
			},
			errContains: "missing token for",
		},
		{
			name: "empty token for placement",
			demands: map[adapter.Key]map[string]any{
				adapter.ZmaticooKey: {
					"token": `{"test-placement-id":{"token":"","timestamp":1700000000000}}`,
				},
			},
			errContains: "empty token for",
		},
		{
			name: "missing timestamp for placement",
			demands: map[adapter.Key]map[string]any{
				adapter.ZmaticooKey: {
					"token": `{"test-placement-id":{"token":"sdk-token"}}`,
				},
			},
			errContains: "missing timestamp for",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auctionRequest := &schema.AuctionRequest{
				AdObject: schema.AdObject{
					Demands: tt.demands,
				},
			}

			_, _, err := zmaticooAdapter.extractTokenAndTimestamp(auctionRequest)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("expected error to contain %q, got %q", tt.errContains, err.Error())
			}
		})
	}
}

func buildAdapter() *ZmaticooAdapter {
	return &ZmaticooAdapter{
		AppID:       "test-app-id",
		PlacementID: "test-placement-id",
	}
}
