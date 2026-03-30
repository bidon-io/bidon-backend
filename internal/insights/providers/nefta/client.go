package nefta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/prebid/openrtb/v19/openrtb2"
)

const (
	DefaultInitURL       = "https://rtb.nefta.app/s2s/init/bidon"
	DefaultFloorPriceURL = "https://rtb.nefta.app/s2s/insight/floor-price/bidon"
	DefaultTimeout       = 2 * time.Second
)

var ErrInvalidNUID = errors.New("nefta init response nuid is empty")

type InitRequest struct {
	NUID        string           `json:"nuid"`
	SessionID   int64            `json:"session_id"`
	AppBundle   string           `json:"app_bundle"`
	AppPlatform string           `json:"app_platform"`
	AppVersion  string           `json:"app_version"`
	SDKVersion  string           `json:"sdk_version"`
	Device      *openrtb2.Device `json:"device,omitempty"`
	UserGeo     *openrtb2.Geo    `json:"user_geo,omitempty"`
}

type InitResponse struct {
	NUID string `json:"nuid"`
}

type Client struct {
	HTTPClient    *http.Client
	InitURL       string
	FloorPriceURL string
	Timeout       time.Duration
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &Client{
		HTTPClient:    httpClient,
		InitURL:       DefaultInitURL,
		FloorPriceURL: DefaultFloorPriceURL,
		Timeout:       DefaultTimeout,
	}
}

type InitCallResult struct {
	Response          InitResponse
	RawRequest        string
	RawRequestHeaders string
	RawResponse       string
	Status            int
}

type FloorPriceRequest struct {
	NUID            string              `json:"nuid"`
	SessionID       int64               `json:"session_id"`
	AppPlatform     string              `json:"app_platform"`
	SDKVersion      string              `json:"sdk_version"`
	AdOpportunityID int64               `json:"ad_opportunity_id"`
	AdType          string              `json:"ad_type"`
	SessionStartTS  int64               `json:"session_start_time"`
	App             *openrtb2.App       `json:"app,omitempty"`
	UserGeo         *openrtb2.Geo       `json:"user_geo,omitempty"`
	Auctions        []FloorPriceAuction `json:"auctions"`
}

type FloorPriceAuction struct {
	ID         int32    `json:"id"`
	FloorPrice float64  `json:"floor_price"`
	Bidders    []string `json:"bidders"`
}

type FloorPriceResponse struct {
	FloorPrices []FloorPriceRecommendation `json:"floor_prices"`
	Control     *bool                      `json:"control,omitempty"`
}

type FloorPriceRecommendation struct {
	AuctionID    int32                  `json:"auction_id"`
	FloorPrice   float64                `json:"floor_price"`
	Accuracy     any                    `json:"accuracy"`
	Notification FloorPriceNotification `json:"notification"`
}

type FloorPriceNotification struct {
	Auction    string `json:"auction"`
	Impression string `json:"impression"`
	Click      string `json:"click"`
}

type FloorPriceCallResult struct {
	Response          FloorPriceResponse
	RawRequest        string
	RawRequestHeaders string
	RawResponse       string
	Status            int
}

func (c *Client) Init(ctx context.Context, req InitRequest) (InitCallResult, error) {
	var result InitCallResult

	payload, err := json.Marshal(req)
	if err != nil {
		return result, fmt.Errorf("marshal nefta init request: %w", err)
	}
	result.RawRequest = string(payload)

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.InitURL, bytes.NewReader(payload))
	if err != nil {
		return result, fmt.Errorf("build nefta init request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("nefta-sdk-version", req.SDKVersion)
	httpReq.Header.Set("nefta-sdk-platform", req.AppPlatform)
	httpReq.Header.Set("nefta-sdk-bundle", req.AppBundle)
	httpReq.Header.Set("nefta-sdk-app-version", req.AppVersion)
	httpReq.Header.Set("nefta-sdk-nuid", req.NUID)
	result.RawRequestHeaders = headerToJSON(httpReq.Header)

	httpResp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return result, fmt.Errorf("send nefta init request: %w", err)
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()
	result.Status = httpResp.StatusCode

	respBody, readErr := io.ReadAll(httpResp.Body)
	if readErr != nil {
		return result, fmt.Errorf("read nefta init response body: %w", readErr)
	}
	result.RawResponse = strings.TrimSpace(string(respBody))

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return result, fmt.Errorf("nefta init request failed: status=%d body=%q", httpResp.StatusCode, result.RawResponse)
	}

	if err = json.Unmarshal(respBody, &result.Response); err != nil {
		return result, fmt.Errorf("decode nefta init response: %w", err)
	}

	if strings.TrimSpace(result.Response.NUID) == "" {
		return result, ErrInvalidNUID
	}

	return result, nil
}

func (c *Client) FloorPrice(ctx context.Context, req FloorPriceRequest) (FloorPriceCallResult, error) {
	var result FloorPriceCallResult

	payload, err := json.Marshal(req)
	if err != nil {
		return result, fmt.Errorf("marshal nefta floor-price request: %w", err)
	}
	result.RawRequest = string(payload)

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.FloorPriceURL, bytes.NewReader(payload))
	if err != nil {
		return result, fmt.Errorf("build nefta floor-price request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("nefta-sdk-version", req.SDKVersion)
	httpReq.Header.Set("nefta-sdk-platform", req.AppPlatform)
	if req.App != nil {
		httpReq.Header.Set("nefta-sdk-bundle", req.App.Bundle)
		httpReq.Header.Set("nefta-sdk-app-version", req.App.Ver)
	}
	httpReq.Header.Set("nefta-sdk-nuid", req.NUID)
	result.RawRequestHeaders = headerToJSON(httpReq.Header)

	httpResp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return result, fmt.Errorf("send nefta floor-price request: %w", err)
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()
	result.Status = httpResp.StatusCode

	respBody, readErr := io.ReadAll(httpResp.Body)
	if readErr != nil {
		return result, fmt.Errorf("read nefta floor-price response body: %w", readErr)
	}
	result.RawResponse = strings.TrimSpace(string(respBody))

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return result, fmt.Errorf("nefta floor-price request failed: status=%d body=%q", httpResp.StatusCode, result.RawResponse)
	}

	if err = json.Unmarshal(respBody, &result.Response); err != nil {
		return result, fmt.Errorf("decode nefta floor-price response: %w", err)
	}

	return result, nil
}

func headerToJSON(header http.Header) string {
	if len(header) == 0 {
		return ""
	}

	payload, err := json.Marshal(header)
	if err != nil {
		return ""
	}

	return string(payload)
}
