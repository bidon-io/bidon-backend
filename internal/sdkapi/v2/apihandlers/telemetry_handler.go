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

	accepted := acceptedCount(req)
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

func acceptedCount(req api.TelemetryRequest) int {
	n := 0
	if req.Events != nil {
		for _, event := range *req.Events {
			if _, err := event.ValueByDiscriminator(); err == nil {
				n++
			}
		}
	}
	if req.Metrics != nil {
		for _, metric := range *req.Metrics {
			if knownMetric(metric.Name) {
				n++
			}
		}
	}
	return n
}

func knownMetric(name api.ClientMetricName) bool {
	switch name {
	case api.ClientAuctionDurationSeconds, api.ClientTokenCollectionDurationSeconds, api.ClientAdLoadDurationSeconds:
		return true
	default:
		return false
	}
}
