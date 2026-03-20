package nefta

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prebid/openrtb/v19/openrtb2"

	"github.com/bidon-io/bidon-backend/internal/insights"
	insightsopenrtb "github.com/bidon-io/bidon-backend/internal/insights/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/bidon-io/bidon-backend/pkg/clock"
)

type initClientMock struct {
	requests []InitRequest
	err      error
}

func (m *initClientMock) Init(_ context.Context, req InitRequest) (InitCallResult, error) {
	m.requests = append(m.requests, req)
	if m.err != nil {
		return InitCallResult{}, m.err
	}
	return InitCallResult{Response: InitResponse{NUID: "nuid-from-nefta"}}, nil
}

type stateStoreMock struct {
	findFn func(ctx context.Context, key string) (*State, error)
	saveFn func(ctx context.Context, key string, state *State) error
}

func (m *stateStoreMock) Find(ctx context.Context, key string) (*State, error) {
	return m.findFn(ctx, key)
}

func (m *stateStoreMock) Save(ctx context.Context, key string, state *State) error {
	return m.saveFn(ctx, key, state)
}

func TestProviderKey(t *testing.T) {
	provider := NewProvider(nil)
	if provider.Key() != insights.NeftaKey {
		t.Fatalf("expected key %q, got %q", insights.NeftaKey, provider.Key())
	}
}

func TestProviderInitMapsRequest(t *testing.T) {
	client := &initClientMock{}
	provider := NewProvider(client)

	_, err := provider.Init(context.Background(), insights.InitRequest{
		NUID:       "existing-nuid",
		SessionID:  7,
		AppVersion: "1.2.3",
		SDKVersion: "0.8.0",
		OpenRTB: insightsopenrtb.InitRequest{
			App:     &openrtb2.App{Bundle: "com.example.app"},
			Device:  &openrtb2.Device{OS: "iOS"},
			UserGeo: &openrtb2.Geo{Country: "USA"},
		},
	})
	if err != nil {
		t.Fatalf("provider init failed: %v", err)
	}

	if len(client.requests) != 1 {
		t.Fatalf("expected one init request, got %d", len(client.requests))
	}

	gotReq := client.requests[0]
	if gotReq.NUID != "existing-nuid" {
		t.Fatalf("expected nuid to be mapped, got %q", gotReq.NUID)
	}
	if gotReq.SessionID != 7 {
		t.Fatalf("expected session_id 7, got %d", gotReq.SessionID)
	}
	if gotReq.AppBundle != "com.example.app" {
		t.Fatalf("expected app_bundle com.example.app, got %q", gotReq.AppBundle)
	}
	if gotReq.AppPlatform != "ios" {
		t.Fatalf("expected app_platform ios, got %q", gotReq.AppPlatform)
	}
	if gotReq.Device == nil || gotReq.Device.OS != "iOS" {
		t.Fatalf("expected device.os iOS, got %+v", gotReq.Device)
	}
	if gotReq.UserGeo == nil || gotReq.UserGeo.Country != "USA" {
		t.Fatalf("expected user_geo.country USA, got %+v", gotReq.UserGeo)
	}
}

func TestProviderInitFallsBackToOpenRTBAppVersion(t *testing.T) {
	client := &initClientMock{}
	provider := NewProvider(client)

	_, err := provider.Init(context.Background(), insights.InitRequest{
		SDKVersion: "0.8.0",
		OpenRTB: insightsopenrtb.InitRequest{
			App: &openrtb2.App{Bundle: "com.example.app", Ver: "9.9.9"},
		},
	})
	if err != nil {
		t.Fatalf("provider init failed: %v", err)
	}

	if len(client.requests) != 1 {
		t.Fatalf("expected one init request, got %d", len(client.requests))
	}
	if client.requests[0].AppVersion != "9.9.9" {
		t.Fatalf("expected fallback app_version 9.9.9, got %q", client.requests[0].AppVersion)
	}
}

func TestProviderInitReturnsClientError(t *testing.T) {
	expectedErr := errors.New("client error")
	client := &initClientMock{err: expectedErr}
	provider := NewProvider(client)

	_, err := provider.Init(context.Background(), insights.InitRequest{})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestProviderInitSkipsWhenAdvertisingIDMissing(t *testing.T) {
	client := &initClientMock{}
	store := &stateStoreMock{
		findFn: func(context.Context, string) (*State, error) {
			t.Fatalf("state store find should not be called when ad id is missing")
			return nil, nil
		},
		saveFn: func(context.Context, string, *State) error {
			t.Fatalf("state store save should not be called when ad id is missing")
			return nil
		},
	}
	provider := NewProvider(client, WithStateStore(store))

	result, err := provider.Init(context.Background(), insights.InitRequest{
		AppID: 1,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Skipped {
		t.Fatalf("expected skipped result when advertising id is missing")
	}
	if len(client.requests) != 0 {
		t.Fatalf("expected no nefta init call, got %d", len(client.requests))
	}
}

func TestProviderInitFirstSession(t *testing.T) {
	client := &initClientMock{}
	mockClock := clock.NewMock()
	mockClock.Set(time.Unix(100, 0))

	var savedKey string
	var savedState *State
	store := &stateStoreMock{
		findFn: func(context.Context, string) (*State, error) {
			return nil, nil
		},
		saveFn: func(_ context.Context, key string, state *State) error {
			savedKey = key
			cp := *state
			savedState = &cp
			return nil
		},
	}
	provider := NewProvider(client, WithStateStore(store), WithClock(mockClock))

	_, err := provider.Init(context.Background(), insights.InitRequest{
		AppID: 11,
		IDFA:  "idfa-1",
		BaseRequest: &schema.BaseRequest{
			Device: schema.Device{OS: "iOS"},
		},
		AppVersion: "1.0.0",
		SDKVersion: "0.9.0",
		OpenRTB: insightsopenrtb.InitRequest{
			App: &openrtb2.App{Bundle: "com.test.app"},
		},
	})
	if err != nil {
		t.Fatalf("provider init failed: %v", err)
	}

	if len(client.requests) != 1 {
		t.Fatalf("expected one nefta init call, got %d", len(client.requests))
	}
	if client.requests[0].NUID != "" || client.requests[0].SessionID != 0 {
		t.Fatalf("expected first session with empty nuid/session=0, got %+v", client.requests[0])
	}
	if savedKey != "nefta:11:idfa-1" {
		t.Fatalf("unexpected state key: %s", savedKey)
	}
	if savedState == nil {
		t.Fatalf("expected state to be saved")
	}
	if savedState.NUID != "nuid-from-nefta" || savedState.SessionID != 0 {
		t.Fatalf("unexpected saved state: %+v", savedState)
	}
}

func TestProviderInitSameSessionSkipsNeftaInitAndRefreshesActivity(t *testing.T) {
	client := &initClientMock{}
	mockClock := clock.NewMock()
	mockClock.Set(time.Unix(1000, 0))

	existing := &State{
		NUID:           "existing-nuid",
		SessionID:      5,
		LastActivityTS: time.Unix(990, 0).UnixMilli(),
		SessionStartTS: time.Unix(900, 0).UnixMilli(),
	}

	var savedState *State
	store := &stateStoreMock{
		findFn: func(context.Context, string) (*State, error) {
			cp := *existing
			return &cp, nil
		},
		saveFn: func(_ context.Context, _ string, state *State) error {
			cp := *state
			savedState = &cp
			return nil
		},
	}
	provider := NewProvider(client, WithStateStore(store), WithClock(mockClock))

	result, err := provider.Init(context.Background(), insights.InitRequest{
		AppID: 12,
		IDG:   "idg-1",
		BaseRequest: &schema.BaseRequest{
			Device: schema.Device{OS: "android"},
		},
	})
	if err != nil {
		t.Fatalf("provider init failed: %v", err)
	}

	if len(client.requests) != 0 {
		t.Fatalf("expected no nefta init call for active session, got %d", len(client.requests))
	}
	if !result.Skipped {
		t.Fatalf("expected skipped result for active session")
	}
	if savedState == nil {
		t.Fatalf("expected state update")
	}
	if savedState.SessionID != existing.SessionID || savedState.NUID != existing.NUID {
		t.Fatalf("session state should be reused, got %+v", savedState)
	}
	if savedState.LastActivityTS != mockClock.Now().UnixMilli() {
		t.Fatalf("last_activity_ts should be refreshed, got %d", savedState.LastActivityTS)
	}
}

func TestProviderInitNewSessionAfterInactivity(t *testing.T) {
	client := &initClientMock{}
	mockClock := clock.NewMock()
	mockClock.Set(time.Unix(2000, 0))

	existing := &State{
		NUID:           "existing-nuid",
		SessionID:      5,
		LastActivityTS: time.Unix(0, 0).UnixMilli(),
		SessionStartTS: time.Unix(900, 0).UnixMilli(),
	}

	var savedState *State
	store := &stateStoreMock{
		findFn: func(context.Context, string) (*State, error) {
			cp := *existing
			return &cp, nil
		},
		saveFn: func(_ context.Context, _ string, state *State) error {
			cp := *state
			savedState = &cp
			return nil
		},
	}
	provider := NewProvider(client, WithStateStore(store), WithClock(mockClock))

	_, err := provider.Init(context.Background(), insights.InitRequest{
		AppID: 13,
		IDFV:  "idfv-1",
		BaseRequest: &schema.BaseRequest{
			Device: schema.Device{OS: "iOS"},
		},
		AppVersion: "2.0.0",
		SDKVersion: "1.0.0",
		OpenRTB: insightsopenrtb.InitRequest{
			App: &openrtb2.App{Bundle: "com.test.app"},
		},
	})
	if err != nil {
		t.Fatalf("provider init failed: %v", err)
	}

	if len(client.requests) != 1 {
		t.Fatalf("expected one nefta init call for new session, got %d", len(client.requests))
	}
	if client.requests[0].NUID != existing.NUID {
		t.Fatalf("expected previous nuid to be reused, got %q", client.requests[0].NUID)
	}
	if client.requests[0].SessionID != existing.SessionID+1 {
		t.Fatalf("expected incremented session_id, got %d", client.requests[0].SessionID)
	}
	if savedState == nil {
		t.Fatalf("expected state update")
	}
	if savedState.SessionID != existing.SessionID+1 {
		t.Fatalf("expected persisted incremented session_id, got %d", savedState.SessionID)
	}
	if savedState.SessionStartTS != mockClock.Now().UnixMilli() {
		t.Fatalf("expected session_start_ts reset for new session, got %d", savedState.SessionStartTS)
	}
}
