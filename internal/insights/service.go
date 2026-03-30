package insights

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrNilProvider       = errors.New("insights provider is nil")
	ErrEmptyProviderKey  = errors.New("insights provider key is empty")
	ErrDuplicateProvider = errors.New("insights provider already registered")
)

type service struct {
	mu                 sync.RWMutex
	providers          map[Key]Provider
	providerSeq        []Key
	initResultFn       func(InitRequest, InitResult)
	floorPriceResultFn func(FloorPriceRequest, FloorPriceResult)
}

type Option func(*service)

func WithInitResultHandler(resultFn func(InitRequest, InitResult)) Option {
	return func(s *service) {
		s.initResultFn = resultFn
	}
}

func WithFloorPriceResultHandler(resultFn func(FloorPriceRequest, FloorPriceResult)) Option {
	return func(s *service) {
		s.floorPriceResultFn = resultFn
	}
}

func NewService(opts ...Option) Service {
	s := &service{
		providers:   make(map[Key]Provider),
		providerSeq: make([]Key, 0, 1),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *service) Register(provider Provider) error {
	if provider == nil {
		return ErrNilProvider
	}

	key := Key(strings.TrimSpace(string(provider.Key())))
	if key == "" {
		return ErrEmptyProviderKey
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.providers[key]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateProvider, key)
	}

	s.providers[key] = provider
	s.providerSeq = append(s.providerSeq, key)

	return nil
}

func (s *service) Init(ctx context.Context, req InitRequest) {
	// Take a thread-safe snapshot of provider registry, so Init can run without
	// holding the lock while providers execute and even if Register happens concurrently.
	s.mu.RLock()
	providerSeq := make([]Key, len(s.providerSeq))
	copy(providerSeq, s.providerSeq)
	providers := make(map[Key]Provider, len(s.providers))
	for key, provider := range s.providers {
		providers[key] = provider
	}
	s.mu.RUnlock()

	if len(providerSeq) == 0 {
		return
	}

	wg := sync.WaitGroup{}

	for _, key := range providerSeq {
		if !isProviderEnabled(req.Settings, key) {
			continue
		}

		provider := providers[key]
		wg.Add(1)
		go func(providerKey Key, providerInstance Provider) {
			defer wg.Done()
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				if s.initResultFn != nil {
					s.initResultFn(req, InitResult{
						Provider: providerKey,
						Error:    fmt.Sprintf("provider panic: %v", recovered),
					})
				}
			}()

			result, err := providerInstance.Init(ctx, req)
			if result.Provider == "" {
				result.Provider = providerKey
			}
			if err != nil && result.Error == "" {
				result.Error = err.Error()
			}
			if s.initResultFn != nil {
				s.initResultFn(req, result)
			}
		}(key, provider)
	}
	wg.Wait()
}

func (s *service) FloorPrice(ctx context.Context, req FloorPriceRequest) []FloorPriceResult {
	// Take a thread-safe snapshot of provider registry, so FloorPrice can run
	// without holding the lock while providers execute.
	s.mu.RLock()
	providerSeq := make([]Key, len(s.providerSeq))
	copy(providerSeq, s.providerSeq)
	providers := make(map[Key]Provider, len(s.providers))
	for key, provider := range s.providers {
		providers[key] = provider
	}
	s.mu.RUnlock()

	if len(providerSeq) == 0 {
		return nil
	}

	var (
		wg      sync.WaitGroup
		resultM sync.Mutex
		results []FloorPriceResult
	)

	for _, key := range providerSeq {
		if !isProviderEnabled(req.Settings, key) {
			continue
		}

		floorPriceProvider, ok := providers[key].(FloorPriceProvider)
		if !ok {
			continue
		}

		wg.Add(1)
		go func(providerKey Key, providerInstance FloorPriceProvider) {
			defer wg.Done()
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				panicResult := FloorPriceResult{
					Provider: providerKey,
					Error:    fmt.Sprintf("provider panic: %v", recovered),
				}
				if s.floorPriceResultFn != nil {
					s.floorPriceResultFn(req, panicResult)
				}
				resultM.Lock()
				results = append(results, panicResult)
				resultM.Unlock()
			}()

			result, err := providerInstance.FloorPrice(ctx, req)
			if result.Provider == "" {
				result.Provider = providerKey
			}
			if err != nil && result.Error == "" {
				result.Error = err.Error()
			}
			if s.floorPriceResultFn != nil {
				s.floorPriceResultFn(req, result)
			}

			resultM.Lock()
			results = append(results, result)
			resultM.Unlock()
		}(key, floorPriceProvider)
	}

	wg.Wait()
	return results
}

func isProviderEnabled(settings map[string]any, key Key) bool {
	if settings == nil {
		return false
	}

	insightsRaw, ok := settings["insights"]
	if !ok {
		return false
	}

	insightsMap, ok := insightsRaw.(map[string]any)
	if !ok {
		return false
	}

	providerRaw, ok := insightsMap[string(key)]
	if !ok {
		return false
	}

	providerMap, ok := providerRaw.(map[string]any)
	if !ok {
		return false
	}

	enabledRaw, ok := providerMap["enabled"]
	if !ok {
		return false
	}

	enabled, ok := enabledRaw.(bool)
	return ok && enabled
}
