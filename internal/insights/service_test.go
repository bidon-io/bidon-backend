package insights

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
)

type testProvider struct {
	key      Key
	initFunc func(context.Context, InitRequest) (InitResult, error)
}

func (p testProvider) Key() Key {
	return p.key
}

func (p testProvider) Init(ctx context.Context, req InitRequest) (InitResult, error) {
	if p.initFunc == nil {
		return InitResult{}, nil
	}
	return p.initFunc(ctx, req)
}

type testFloorPriceProvider struct {
	key            Key
	initFunc       func(context.Context, InitRequest) (InitResult, error)
	floorPriceFunc func(context.Context, FloorPriceRequest) (FloorPriceResult, error)
}

func (p testFloorPriceProvider) Key() Key {
	return p.key
}

func (p testFloorPriceProvider) Init(ctx context.Context, req InitRequest) (InitResult, error) {
	if p.initFunc == nil {
		return InitResult{}, nil
	}
	return p.initFunc(ctx, req)
}

func (p testFloorPriceProvider) FloorPrice(ctx context.Context, req FloorPriceRequest) (FloorPriceResult, error) {
	if p.floorPriceFunc == nil {
		return FloorPriceResult{}, nil
	}
	return p.floorPriceFunc(ctx, req)
}

func TestServiceRegister(t *testing.T) {
	svc := NewService()

	if err := svc.Register(nil); !errors.Is(err, ErrNilProvider) {
		t.Fatalf("expected ErrNilProvider, got: %v", err)
	}

	if err := svc.Register(testProvider{}); !errors.Is(err, ErrEmptyProviderKey) {
		t.Fatalf("expected ErrEmptyProviderKey, got: %v", err)
	}

	if err := svc.Register(testProvider{key: NeftaKey}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	if err := svc.Register(testProvider{key: NeftaKey}); !errors.Is(err, ErrDuplicateProvider) {
		t.Fatalf("expected ErrDuplicateProvider, got: %v", err)
	}
}

func TestServiceInitIsolatesProviderErrors(t *testing.T) {
	var (
		callMu sync.Mutex
		calls  []string
	)

	svc := NewService()

	err := svc.Register(testProvider{
		key: "failing-provider",
		initFunc: func(context.Context, InitRequest) (InitResult, error) {
			callMu.Lock()
			calls = append(calls, "failing-provider")
			callMu.Unlock()
			return InitResult{}, errors.New("boom")
		},
	})
	if err != nil {
		t.Fatalf("register failing provider: %v", err)
	}

	err = svc.Register(testProvider{
		key: "healthy-provider",
		initFunc: func(context.Context, InitRequest) (InitResult, error) {
			callMu.Lock()
			calls = append(calls, "healthy-provider")
			callMu.Unlock()
			return InitResult{}, nil
		},
	})
	if err != nil {
		t.Fatalf("register healthy provider: %v", err)
	}

	svc.Init(context.Background(), InitRequest{
		AppID: 1,
		Settings: map[string]any{
			"insights": map[string]any{
				"failing-provider": map[string]any{"enabled": true},
				"healthy-provider": map[string]any{"enabled": true},
			},
		},
	})

	callMu.Lock()
	gotCalls := append([]string(nil), calls...)
	callMu.Unlock()

	if len(gotCalls) != 2 {
		t.Fatalf("expected 2 providers to execute, got %d", len(gotCalls))
	}

	if !slices.Contains(gotCalls, "failing-provider") {
		t.Fatalf("expected failing provider to be called, calls: %v", gotCalls)
	}

	if !slices.Contains(gotCalls, "healthy-provider") {
		t.Fatalf("expected healthy provider to be called, calls: %v", gotCalls)
	}
}

func TestServiceInitSkipsDisabledProviders(t *testing.T) {
	var calls []string
	svc := NewService()

	err := svc.Register(testProvider{
		key: "enabled-provider",
		initFunc: func(context.Context, InitRequest) (InitResult, error) {
			calls = append(calls, "enabled-provider")
			return InitResult{}, nil
		},
	})
	if err != nil {
		t.Fatalf("register enabled provider: %v", err)
	}

	err = svc.Register(testProvider{
		key: "disabled-provider",
		initFunc: func(context.Context, InitRequest) (InitResult, error) {
			calls = append(calls, "disabled-provider")
			return InitResult{}, nil
		},
	})
	if err != nil {
		t.Fatalf("register disabled provider: %v", err)
	}

	svc.Init(context.Background(), InitRequest{
		Settings: map[string]any{
			"insights": map[string]any{
				"enabled-provider":  map[string]any{"enabled": true},
				"disabled-provider": map[string]any{"enabled": false},
			},
		},
	})

	if len(calls) != 1 || calls[0] != "enabled-provider" {
		t.Fatalf("expected only enabled provider to be called, got %v", calls)
	}
}

func TestServiceInitRecoversProviderPanic(t *testing.T) {
	var (
		mu      sync.Mutex
		results []InitResult
	)

	svc := NewService(WithInitResultHandler(func(_ InitRequest, result InitResult) {
		mu.Lock()
		defer mu.Unlock()
		results = append(results, result)
	}))

	err := svc.Register(testProvider{
		key: "panic-provider",
		initFunc: func(context.Context, InitRequest) (InitResult, error) {
			panic("boom")
		},
	})
	if err != nil {
		t.Fatalf("register panic provider: %v", err)
	}

	err = svc.Register(testProvider{
		key: "healthy-provider",
		initFunc: func(context.Context, InitRequest) (InitResult, error) {
			return InitResult{Status: 200}, nil
		},
	})
	if err != nil {
		t.Fatalf("register healthy provider: %v", err)
	}

	svc.Init(context.Background(), InitRequest{
		Settings: map[string]any{
			"insights": map[string]any{
				"panic-provider":   map[string]any{"enabled": true},
				"healthy-provider": map[string]any{"enabled": true},
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()

	if len(results) != 2 {
		t.Fatalf("expected 2 init results, got %d", len(results))
	}

	var panicResult *InitResult
	var healthyResult *InitResult
	for i := range results {
		result := results[i]
		switch result.Provider {
		case "panic-provider":
			panicResult = &result
		case "healthy-provider":
			healthyResult = &result
		}
	}

	if panicResult == nil {
		t.Fatalf("expected panic-provider result, got %+v", results)
	}
	if panicResult.Error == "" {
		t.Fatalf("expected panic-provider error to be set")
	}
	if panicResult.Error != fmt.Sprintf("provider panic: %v", "boom") {
		t.Fatalf("unexpected panic error: %q", panicResult.Error)
	}

	if healthyResult == nil {
		t.Fatalf("expected healthy-provider result, got %+v", results)
	}
	if healthyResult.Error != "" {
		t.Fatalf("expected healthy-provider to have no error, got %q", healthyResult.Error)
	}
	if healthyResult.Status != 200 {
		t.Fatalf("expected healthy-provider status 200, got %d", healthyResult.Status)
	}
}

func TestServiceFloorPriceUsesOnlyFloorPriceProviders(t *testing.T) {
	svc := NewService()

	var (
		mu                sync.Mutex
		initOnlyCallCount int
		floorCallCount    int
	)

	err := svc.Register(testProvider{
		key: "init-only-provider",
		initFunc: func(context.Context, InitRequest) (InitResult, error) {
			mu.Lock()
			initOnlyCallCount++
			mu.Unlock()
			return InitResult{}, nil
		},
	})
	if err != nil {
		t.Fatalf("register init-only provider: %v", err)
	}

	err = svc.Register(testFloorPriceProvider{
		key: "floor-price-provider",
		floorPriceFunc: func(context.Context, FloorPriceRequest) (FloorPriceResult, error) {
			mu.Lock()
			floorCallCount++
			mu.Unlock()
			return FloorPriceResult{
				Auction: &FloorPriceRecommendation{AuctionID: 1, FloorPrice: 1.23},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("register floor-price provider: %v", err)
	}

	results := svc.FloorPrice(context.Background(), FloorPriceRequest{
		Settings: map[string]any{
			"insights": map[string]any{
				"init-only-provider":   map[string]any{"enabled": true},
				"floor-price-provider": map[string]any{"enabled": true},
			},
		},
	})

	mu.Lock()
	defer mu.Unlock()

	if initOnlyCallCount != 0 {
		t.Fatalf("expected init-only provider to be skipped for FloorPrice, got %d calls", initOnlyCallCount)
	}
	if floorCallCount != 1 {
		t.Fatalf("expected floor-price provider to be called once, got %d", floorCallCount)
	}
	if len(results) != 1 {
		t.Fatalf("expected one floor-price result, got %d", len(results))
	}
	if results[0].Provider != "floor-price-provider" {
		t.Fatalf("expected provider floor-price-provider, got %q", results[0].Provider)
	}
}

func TestServiceFloorPriceSkipsDisabledProviders(t *testing.T) {
	svc := NewService()

	err := svc.Register(testFloorPriceProvider{
		key: "floor-price-provider",
		floorPriceFunc: func(context.Context, FloorPriceRequest) (FloorPriceResult, error) {
			return FloorPriceResult{
				Auction: &FloorPriceRecommendation{AuctionID: 7, FloorPrice: 0.77},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("register floor-price provider: %v", err)
	}

	results := svc.FloorPrice(context.Background(), FloorPriceRequest{
		Settings: map[string]any{
			"insights": map[string]any{
				"floor-price-provider": map[string]any{"enabled": false},
			},
		},
	})

	if len(results) != 0 {
		t.Fatalf("expected no results for disabled provider, got %d", len(results))
	}
}

func TestServiceFloorPriceInvokesResultHandlerForSuccessAndPanic(t *testing.T) {
	var (
		mu      sync.Mutex
		results []FloorPriceResult
	)

	svc := NewService(WithFloorPriceResultHandler(func(_ FloorPriceRequest, result FloorPriceResult) {
		mu.Lock()
		defer mu.Unlock()
		results = append(results, result)
	}))

	err := svc.Register(testFloorPriceProvider{
		key: "panic-provider",
		floorPriceFunc: func(context.Context, FloorPriceRequest) (FloorPriceResult, error) {
			panic("boom")
		},
	})
	if err != nil {
		t.Fatalf("register panic provider: %v", err)
	}

	err = svc.Register(testFloorPriceProvider{
		key: "healthy-provider",
		floorPriceFunc: func(context.Context, FloorPriceRequest) (FloorPriceResult, error) {
			return FloorPriceResult{
				Auction: &FloorPriceRecommendation{AuctionID: 1, FloorPrice: 0.55},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("register healthy provider: %v", err)
	}

	got := svc.FloorPrice(context.Background(), FloorPriceRequest{
		Settings: map[string]any{
			"insights": map[string]any{
				"panic-provider":   map[string]any{"enabled": true},
				"healthy-provider": map[string]any{"enabled": true},
			},
		},
	})

	if len(got) != 2 {
		t.Fatalf("expected 2 floor-price results, got %d", len(got))
	}

	mu.Lock()
	defer mu.Unlock()

	if len(results) != 2 {
		t.Fatalf("expected 2 handler callbacks, got %d", len(results))
	}
}
