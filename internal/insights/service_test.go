package insights

import (
	"context"
	"errors"
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
