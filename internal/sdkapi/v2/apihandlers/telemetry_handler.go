package apihandlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime/types"
	"go.uber.org/zap"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/api"
)

const (
	telemetryMaxBytes   = 256 << 10
	telemetryMaxEvents  = 100
	telemetryMaxMetrics = 50
)

type TelemetryHandler struct {
	Logger *zap.Logger
}

func (h *TelemetryHandler) Handle(c echo.Context) error {
	body := http.MaxBytesReader(c.Response(), c.Request().Body, telemetryMaxBytes)
	defer body.Close()

	raw, err := io.ReadAll(body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "telemetry batch too large")
		}
		return echo.NewHTTPError(http.StatusBadRequest, "malformed telemetry batch")
	}

	var req api.TelemetryRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "malformed telemetry batch")
	}
	if req.App.Key == "" || req.App.Bundle == "" || req.Session.Id == (types.UUID{}) {
		return echo.NewHTTPError(http.StatusBadRequest, "telemetry request requires app.key, app.bundle, and session.id")
	}
	if eventCount(req) > telemetryMaxEvents || metricCount(req) > telemetryMaxMetrics {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "telemetry batch too large")
	}

	accepted := acceptedCount(raw)
	h.Logger.Info("telemetry ingest",
		zap.String("sdk_version", c.Request().Header.Get("X-Bidon-Version")),
		zap.String("app_key", req.App.Key),
		zap.Int("bytes", len(raw)),
		zap.Int("accepted", accepted),
		zap.ByteString("body", raw),
	)

	return c.JSON(http.StatusAccepted, api.TelemetryResponse{Accepted: accepted})
}

func eventCount(req api.TelemetryRequest) int {
	if req.Events == nil {
		return 0
	}
	return len(*req.Events)
}

func metricCount(req api.TelemetryRequest) int {
	if req.Metrics == nil {
		return 0
	}
	return len(*req.Metrics)
}

func acceptedCount(raw []byte) int {
	var body struct {
		Events []struct {
			Envelope struct {
				EventName string `json:"event_name"`
			} `json:"envelope"`
		} `json:"events"`
		Metrics []struct {
			Name api.ClientMetricName `json:"name"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return 0
	}

	n := 0
	for _, event := range body.Events {
		if _, ok := clientEventNames[event.Envelope.EventName]; ok {
			n++
		}
	}
	for _, metric := range body.Metrics {
		if knownMetric(metric.Name) {
			n++
		}
	}
	return n
}

// clientEventNames is the PRD client catalogue on POST /v2/telemetry.
// Server-minted auction, DSP, and notice names are absent on purpose.
var clientEventNames = map[string]struct{}{
	"sdk_init_started":             {},
	"sdk_init_completed":           {},
	"sdk_init_failed":              {},
	"adapter_init_result":          {},
	"config_fetch_result":          {},
	"adapter_token_requested":      {},
	"adapter_token_result":         {},
	"token_collection_completed":   {},
	"ad_request_started":           {},
	"auction_requested":            {},
	"auction_response_received":    {},
	"ad_filled":                    {},
	"auction_failed":               {},
	"auction_no_demand":            {},
	"ad_load_requested":            {},
	"adm_parse_result":             {},
	"renderer_selected":            {},
	"ad_loaded":                    {},
	"ad_load_failed":               {},
	"ad_expired":                   {},
	"ad_show_requested":            {},
	"ad_impression":                {},
	"ad_show_failed":               {},
	"ad_viewable":                  {},
	"ad_clicked":                   {},
	"ad_closed":                    {},
	"ad_reward_granted":            {},
	"webview_error":                {},
	"render_crashed":               {},
	"video_start":                  {},
	"video_q1":                     {},
	"video_midpoint":               {},
	"video_q3":                     {},
	"video_complete":               {},
	"video_skipped":                {},
	"video_error":                  {},
	"adapter_load_budget_exceeded": {},
	"no_fill_returned_to_max":      {},
	"signal_requested":             {},
	"signal_provided":              {},
	"signal_failed":                {},
	"signal_timeout_suspected":     {},
}

func knownMetric(name api.ClientMetricName) bool {
	switch name {
	case api.ClientAuctionDurationSeconds, api.ClientTokenCollectionDurationSeconds, api.ClientAdLoadDurationSeconds:
		return true
	default:
		return false
	}
}
