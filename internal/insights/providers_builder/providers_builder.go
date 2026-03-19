package providers_builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/bidon-io/bidon-backend/internal/insights"
	"github.com/bidon-io/bidon-backend/internal/insights/providers/nefta"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
)

var ErrNilRedisClient = errors.New("insights providers builder: redis client is nil")

const insightsProviderInitEventTypePrefix = "insights_provider_init"

type Deps struct {
	Redis       *redis.ClusterClient
	EventLogger *event.Logger
	HTTPClient  *http.Client
}

func Build(deps Deps) (insights.Service, error) {
	if deps.Redis == nil {
		return nil, ErrNilRedisClient
	}

	service := insights.NewService(insights.WithInitResultHandler(newInitResultLogger(deps.EventLogger)))
	neftaClient := nefta.NewClient(deps.HTTPClient)

	err := service.Register(nefta.NewProvider(
		neftaClient,
		nefta.WithStateStore(nefta.NewRedisStateStore(deps.Redis)),
	))
	if err != nil {
		return nil, err
	}

	return service, nil
}

func newInitResultLogger(eventLogger *event.Logger) func(insights.InitRequest, insights.InitResult) {
	return func(req insights.InitRequest, result insights.InitResult) {
		if eventLogger == nil {
			return
		}
		if req.BaseRequest == nil {
			return
		}

		adRequestParams := event.AdRequestParams{
			EventType:   insightsProviderInitEventType(result.Provider),
			Status:      initResultStatus(result),
			RawRequest:  combineRawWithHeaders(result.RawRequest, result.RawRequestHeaders),
			RawResponse: result.RawResponse,
			Error:       result.Error,
		}
		ev := event.NewAdEvent(req.BaseRequest, adRequestParams, req.GeoData)

		eventLogger.Log(ev, func(err error) {
			log.Printf("log insights init event: %v", err)
		})
	}
}

func insightsProviderInitEventType(providerKey insights.Key) string {
	if providerKey == "" {
		return insightsProviderInitEventTypePrefix
	}
	return fmt.Sprintf("%s_%s", insightsProviderInitEventTypePrefix, providerKey)
}

func initResultStatus(result insights.InitResult) string {
	switch {
	case result.Error != "":
		return "ERROR"
	case result.Skipped:
		return "SKIPPED"
	case result.Status > 0:
		return "SUCCESS"
	default:
		return ""
	}
}

func combineRawWithHeaders(body, headers string) string {
	if headers == "" {
		return body
	}

	payload := map[string]any{
		"headers": decodeJSONOrRaw(headers),
	}

	if body != "" {
		payload["body"] = decodeJSONOrRaw(body)
	}

	merged, err := json.Marshal(payload)
	if err != nil {
		return body
	}

	return string(merged)
}

func decodeJSONOrRaw(raw string) any {
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
		return decoded
	}

	return raw
}
