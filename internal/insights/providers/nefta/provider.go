package nefta

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bidon-io/bidon-backend/internal/insights"
	"github.com/bidon-io/bidon-backend/pkg/clock"
	"github.com/prebid/openrtb/v19/openrtb2"
)

type initClient interface {
	Init(ctx context.Context, req InitRequest) (InitCallResult, error)
}

type floorPriceClient interface {
	FloorPrice(ctx context.Context, req FloorPriceRequest) (FloorPriceCallResult, error)
}

const SessionInactivityTimeout = 30 * time.Minute

type Option func(*Provider)

func WithStateStore(store StateStore) Option {
	return func(p *Provider) {
		p.stateStore = store
	}
}

func WithClock(c clock.Clock) Option {
	return func(p *Provider) {
		p.clock = c
	}
}

func WithFloorPriceClient(client floorPriceClient) Option {
	return func(p *Provider) {
		p.floorPriceClient = client
	}
}

type Provider struct {
	client                   initClient
	floorPriceClient         floorPriceClient
	stateStore               StateStore
	clock                    clock.Clock
	sessionInactivityTimeout time.Duration
}

type sessionStateResult struct {
	State          *State
	InitCallResult *InitCallResult
	Skipped        bool
}

func NewProvider(client initClient, opts ...Option) *Provider {
	if client == nil {
		client = NewClient(nil)
	}

	p := &Provider{
		client:                   client,
		clock:                    clock.New(),
		sessionInactivityTimeout: SessionInactivityTimeout,
	}
	if c, ok := client.(floorPriceClient); ok {
		p.floorPriceClient = c
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Provider) Key() insights.Key {
	return insights.NeftaKey
}

func (p *Provider) Init(ctx context.Context, req insights.InitRequest) (insights.InitResult, error) {
	result := insights.InitResult{Provider: insights.NeftaKey}

	if p.stateStore == nil {
		initResult, err := p.client.Init(ctx, toNeftaInitRequest(req))
		// Preserve raw payloads/status for observability even when Init returns an error.
		result.RawRequest = initResult.RawRequest
		result.RawRequestHeaders = initResult.RawRequestHeaders
		result.RawResponse = initResult.RawResponse
		result.Status = initResult.Status
		return result, err
	}

	advertisingID := insights.ResolveAdvertisingID(req)
	if advertisingID == "" {
		result.Skipped = true
		return result, nil
	}

	key := stateKey(req.AppID, advertisingID)
	sessionResult, err := p.ensureSessionState(ctx, key, req, true)
	if sessionResult.InitCallResult != nil {
		// Preserve raw payloads/status for observability even when Init returns an error.
		result.RawRequest = sessionResult.InitCallResult.RawRequest
		result.RawRequestHeaders = sessionResult.InitCallResult.RawRequestHeaders
		result.RawResponse = sessionResult.InitCallResult.RawResponse
		result.Status = sessionResult.InitCallResult.Status
	}
	if err != nil {
		return result, err
	}

	result.Skipped = sessionResult.Skipped
	return result, nil
}

func (p *Provider) FloorPrice(ctx context.Context, req insights.FloorPriceRequest) (insights.FloorPriceResult, error) {
	result := insights.FloorPriceResult{Provider: insights.NeftaKey}

	if p.floorPriceClient == nil || p.stateStore == nil {
		result.Skipped = true
		return result, nil
	}

	if !isNeftaAdTypeSupported(req.AdType) {
		result.Skipped = true
		return result, nil
	}

	advertisingID := resolveFloorPriceAdvertisingID(req)
	if advertisingID == "" {
		result.Skipped = true
		return result, nil
	}

	key := stateKey(req.AppID, advertisingID)
	sessionResult, err := p.ensureSessionState(ctx, key, insights.InitRequest{
		AppID:      req.AppID,
		AppVersion: req.AppVersion,
		SDKVersion: req.SDKVersion,
		OpenRTB:    req.OpenRTB,
	}, false)
	if err != nil {
		return result, err
	}
	state := sessionResult.State
	if state == nil || strings.TrimSpace(state.NUID) == "" {
		result.Skipped = true
		return result, nil
	}

	nextAdOpportunityID := state.AdOpportunityID + 1
	if nextAdOpportunityID <= 0 {
		nextAdOpportunityID = 1
	}

	callResult, callErr := p.floorPriceClient.FloorPrice(ctx, toNeftaFloorPriceRequest(req, state, nextAdOpportunityID))
	result.RawRequest = callResult.RawRequest
	result.RawRequestHeaders = callResult.RawRequestHeaders
	result.RawResponse = callResult.RawResponse
	result.Status = callResult.Status
	if callResult.Response.Control != nil {
		control := *callResult.Response.Control
		result.Control = &control
	}
	result.Auction = mapFloorPriceAuction(callResult.Response.FloorPrices)

	// Control group: zero out floor price but preserve notification links.
	// Nefta requires requests and notifications to still be sent for uplift calculation.
	if result.Control != nil && *result.Control && result.Auction != nil {
		result.Auction.FloorPrice = 0
	}

	nowTS := p.clock.Now().UnixMilli()
	state.AdOpportunityID = nextAdOpportunityID
	state.LastActivityTS = nowTS
	if saveErr := p.stateStore.Save(ctx, key, state); saveErr != nil {
		if callErr != nil {
			return result, callErr
		}
		return result, saveErr
	}

	return result, callErr
}

func (p *Provider) ensureSessionState(
	ctx context.Context,
	key string,
	req insights.InitRequest,
	refreshLastActivityOnSkipped bool,
) (sessionStateResult, error) {
	sessionResult := sessionStateResult{}
	nowTS := p.clock.Now().UnixMilli()

	state, err := p.stateStore.Find(ctx, key)
	if err != nil {
		return sessionResult, err
	}

	if state == nil {
		initCallResult, callErr := p.client.Init(ctx, toNeftaInitRequest(insights.InitRequest{
			AppID:      req.AppID,
			NUID:       "",
			SessionID:  0,
			AppVersion: req.AppVersion,
			SDKVersion: req.SDKVersion,
			OpenRTB:    req.OpenRTB,
		}))
		sessionResult.InitCallResult = &initCallResult
		if callErr != nil {
			return sessionResult, callErr
		}

		state = &State{
			NUID:            initCallResult.Response.NUID,
			SessionID:       0,
			AdOpportunityID: 0,
			LastActivityTS:  nowTS,
			SessionStartTS:  nowTS,
		}
		if saveErr := p.stateStore.Save(ctx, key, state); saveErr != nil {
			return sessionResult, saveErr
		}
		sessionResult.State = state
		return sessionResult, nil
	}

	lastActivity := time.UnixMilli(state.LastActivityTS)
	if p.clock.Since(lastActivity) > p.sessionInactivityTimeout {
		nextSessionID := state.SessionID + 1
		initCallResult, callErr := p.client.Init(ctx, toNeftaInitRequest(insights.InitRequest{
			AppID:      req.AppID,
			NUID:       state.NUID,
			SessionID:  nextSessionID,
			AppVersion: req.AppVersion,
			SDKVersion: req.SDKVersion,
			OpenRTB:    req.OpenRTB,
		}))
		sessionResult.InitCallResult = &initCallResult
		if callErr != nil {
			return sessionResult, callErr
		}

		state = &State{
			NUID:            initCallResult.Response.NUID,
			SessionID:       nextSessionID,
			AdOpportunityID: 0,
			LastActivityTS:  nowTS,
			SessionStartTS:  nowTS,
		}
		if saveErr := p.stateStore.Save(ctx, key, state); saveErr != nil {
			return sessionResult, saveErr
		}
		sessionResult.State = state
		return sessionResult, nil
	}

	if refreshLastActivityOnSkipped {
		state.LastActivityTS = nowTS
		if saveErr := p.stateStore.Save(ctx, key, state); saveErr != nil {
			return sessionResult, saveErr
		}
	}
	sessionResult.State = state
	sessionResult.Skipped = true
	return sessionResult, nil
}

func stateKey(appID int64, deviceAdvertisingID string) string {
	return fmt.Sprintf("nefta:%d:%s", appID, deviceAdvertisingID)
}

func toNeftaInitRequest(req insights.InitRequest) InitRequest {
	appVersion := strings.TrimSpace(req.AppVersion)
	if appVersion == "" && req.OpenRTB.App != nil {
		appVersion = req.OpenRTB.App.Ver
	}

	appBundle := ""
	if req.OpenRTB.App != nil {
		appBundle = req.OpenRTB.App.Bundle
	}

	return InitRequest{
		NUID:        req.NUID,
		SessionID:   req.SessionID,
		AppBundle:   appBundle,
		AppPlatform: resolveAppPlatform(req.OpenRTB.Device),
		AppVersion:  appVersion,
		SDKVersion:  req.SDKVersion,
		Device:      req.OpenRTB.Device,
		UserGeo:     req.OpenRTB.UserGeo,
	}
}

func toNeftaFloorPriceRequest(req insights.FloorPriceRequest, state *State, adOpportunityID int64) FloorPriceRequest {
	return FloorPriceRequest{
		NUID:            state.NUID,
		SessionID:       state.SessionID,
		AppPlatform:     resolveAppPlatform(req.OpenRTB.Device),
		SDKVersion:      req.SDKVersion,
		AdOpportunityID: adOpportunityID,
		AdType:          req.AdType,
		SessionStartTS:  state.SessionStartTS,
		App:             req.OpenRTB.App,
		UserGeo:         req.OpenRTB.UserGeo,
		Auctions: []FloorPriceAuction{
			{
				ID:         1,
				FloorPrice: req.FloorPrice,
				Bidders:    req.Bidders,
			},
		},
	}
}

func mapFloorPriceAuction(auctions []FloorPriceRecommendation) *insights.FloorPriceRecommendation {
	if len(auctions) == 0 {
		return nil
	}

	best := auctions[0]
	for _, auction := range auctions[1:] {
		if auction.FloorPrice > best.FloorPrice {
			best = auction
		}
	}

	return &insights.FloorPriceRecommendation{
		AuctionID:  best.AuctionID,
		FloorPrice: best.FloorPrice,
		Accuracy:   best.Accuracy,
		Notification: insights.FloorPriceNotification{
			Auction:    best.Notification.Auction,
			Impression: best.Notification.Impression,
			Click:      best.Notification.Click,
		},
	}
}

func resolveFloorPriceAdvertisingID(req insights.FloorPriceRequest) string {
	if req.BaseRequest == nil {
		return ""
	}

	switch req.BaseRequest.Device.OS {
	case "iOS":
		if req.IDFA != "" {
			return req.IDFA
		}
		return req.IDFV
	case "android":
		return req.IDG
	default:
		return ""
	}
}

func isNeftaAdTypeSupported(adType string) bool {
	return adType == "interstitial" || adType == "rewarded"
}

func resolveAppPlatform(device *openrtb2.Device) string {
	if device == nil {
		return ""
	}

	platform := strings.ToLower(strings.TrimSpace(device.OS))
	if platform == "ios" || platform == "android" {
		return platform
	}

	return ""
}
