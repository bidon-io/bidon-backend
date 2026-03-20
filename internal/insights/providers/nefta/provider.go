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

type Provider struct {
	client                   initClient
	stateStore               StateStore
	clock                    clock.Clock
	sessionInactivityTimeout time.Duration
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
	nowTS := p.clock.Now().UnixMilli()

	state, err := p.stateStore.Find(ctx, key)
	if err != nil {
		return result, err
	}

	if state == nil {
		initCallResult, err := p.client.Init(ctx, toNeftaInitRequest(insights.InitRequest{
			AppID:      req.AppID,
			NUID:       "",
			SessionID:  0,
			AppVersion: req.AppVersion,
			SDKVersion: req.SDKVersion,
			OpenRTB:    req.OpenRTB,
		}))
		// Preserve raw payloads/status for observability even when Init returns an error.
		result.RawRequest = initCallResult.RawRequest
		result.RawRequestHeaders = initCallResult.RawRequestHeaders
		result.RawResponse = initCallResult.RawResponse
		result.Status = initCallResult.Status
		if err != nil {
			return result, err
		}

		err = p.stateStore.Save(ctx, key, &State{
			NUID:           initCallResult.Response.NUID,
			SessionID:      0,
			LastActivityTS: nowTS,
			SessionStartTS: nowTS,
		})
		return result, err
	}

	lastActivity := time.UnixMilli(state.LastActivityTS)
	if p.clock.Since(lastActivity) > p.sessionInactivityTimeout {
		nextSessionID := state.SessionID + 1
		initCallResult, err := p.client.Init(ctx, toNeftaInitRequest(insights.InitRequest{
			AppID:      req.AppID,
			NUID:       state.NUID,
			SessionID:  nextSessionID,
			AppVersion: req.AppVersion,
			SDKVersion: req.SDKVersion,
			OpenRTB:    req.OpenRTB,
		}))
		// Preserve raw payloads/status for observability even when Init returns an error.
		result.RawRequest = initCallResult.RawRequest
		result.RawRequestHeaders = initCallResult.RawRequestHeaders
		result.RawResponse = initCallResult.RawResponse
		result.Status = initCallResult.Status
		if err != nil {
			return result, err
		}

		err = p.stateStore.Save(ctx, key, &State{
			NUID:           initCallResult.Response.NUID,
			SessionID:      nextSessionID,
			LastActivityTS: nowTS,
			SessionStartTS: nowTS,
		})
		return result, err
	}

	result.Skipped = true
	state.LastActivityTS = nowTS
	err = p.stateStore.Save(ctx, key, state)
	return result, err
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
