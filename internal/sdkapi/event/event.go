package event

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/maps"
)

type Event struct {
	Topic   Topic
	Payload map[string]any
}

type Topic string

const (
	UnknownTopic Topic = "unknown"
	ConfigTopic  Topic = "config"
)

type RequestMapper interface {
	Map() map[string]any
}

// New creates new Event for Logger to log.
// TODO: We need to add application specific `context`, because we need to be able
// to log error to Sentry from different parts of application, and right now we can't do this without echo.Context
func New(c echo.Context, topic Topic, mapper RequestMapper, geoData geocoder.GeoData) Event {
	payload := mapper.Map()

	payload["timestamp"] = float64(time.Now().UnixNano()) / 1e9
	payload["geo"] = eventGeo(payload["geo"], geoData)

	ext, err := eventExt(payload["ext"])
	if err != nil {
		logError(c, err)
	}

	payload["ext"] = ext

	return Event{
		Topic:   topic,
		Payload: payload,
	}
}

func eventGeo(geo any, geoData geocoder.GeoData) map[string]any {
	var eventGeo map[string]any
	if geoData != (geocoder.GeoData{}) {
		eventGeo = map[string]any{
			"ip":         geoData.IPString,
			"country":    geoData.CountryCode,
			"country_id": geoData.CountryID,
		}
	} else {
		eventGeo = make(map[string]any)
	}

	payloadGeo, _ := geo.(map[string]any)
	if payloadGeo == nil {
		return eventGeo
	}

	maps.Copy(payloadGeo, eventGeo)

	return payloadGeo
}

func eventExt(ext any) (map[string]any, error) {
	eventExt := make(map[string]any)

	payloadExt, ok := ext.(string)
	if !ok || payloadExt == "" {
		return eventExt, nil
	}

	err := json.Unmarshal([]byte(payloadExt), &eventExt)
	if err != nil {
		return eventExt, fmt.Errorf("unmarshal ext: %v", err)
	}

	return eventExt, nil
}

func logError(c echo.Context, err error) {
	c.Logger().Error(err)

	hub := sentryecho.GetHubFromContext(c)
	if hub != nil {
		client, scope := hub.Client(), hub.Scope()
		client.CaptureException(
			err,
			&sentry.EventHint{Context: c.Request().Context()},
			scope,
		)
	}
}
