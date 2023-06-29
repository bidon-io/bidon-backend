package sdkapi

import (
	"context"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
	"github.com/bidon-io/bidon-backend/internal/segment"
	"os"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/labstack/echo/v4"
)

// BaseHandler provides common functionality between sdkapi handlers
type BaseHandler struct {
	AppFetcher AppFetcher
	Geocoder   Geocoder
}

type AppFetcher interface {
	Fetch(ctx context.Context, appKey, appBundle string) (*App, error)
}

type SegmentFetcher interface {
	Fetch(ctx context.Context, params *segment.Params) (segment.Segment, error)
}

type Geocoder interface {
	FindGeoData(ctx context.Context, ipString string) (*geocoder.GeoData, error)
}

func (b *BaseHandler) resolveRequest(c echo.Context) (*request, error) {
	var raw schema.Request
	if err := c.Bind(&raw); err != nil {
		return nil, err
	}

	app, err := b.AppFetcher.Fetch(c.Request().Context(), raw.App.Key, raw.App.Bundle)
	if err != nil {
		return nil, err
	}

	var country *Country
	if os.Getenv("USE_GEOCODING") == "true" {
		geoData, err := b.Geocoder.FindGeoData(c.Request().Context(), c.RealIP())
		if err != nil {
			return nil, err
		}
		country = &Country{ID: geoData.CountryID, CountryCode: geoData.CountryCode}
	} else {
		country = &Country{ID: 0, CountryCode: raw.Geo.Country}
	}

	return &request{
		raw:     raw,
		app:     app,
		country: country,
	}, nil
}
