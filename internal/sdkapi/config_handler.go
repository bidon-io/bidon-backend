package sdkapi

import (
	"github.com/bidon-io/bidon-backend/internal/segment"
	"net/http"

	"github.com/bidon-io/bidon-backend/internal/config"
	"github.com/labstack/echo/v4"
)

type ConfigHandler struct {
	*BaseHandler
	AdaptersBuilder *config.AdaptersBuilder
}

type ConfigResponse struct {
	Init       ConfigResponseInit `json:"init"`
	Placements []any              `json:"placements"`
	Token      string             `json:"token"`
	Segment    Segment            `json:"segment"`
}

type Segment struct {
	ID *int64 `json:"id"`
}

type ConfigResponseInit struct {
	TMax     int             `json:"tmax"`
	Adapters config.Adapters `json:"adapters"`
}

func (h *ConfigHandler) Handle(c echo.Context) error {
	req, err := h.resolveRequest(c)
	if err != nil {
		return err
	}

	country, _ := h.OfflineGeocoder.FindGeoData(c.Request().Context(), c.RealIP())

	segmentParams := &segment.Params{
		Country: country.CountryCode,
		Ext:     req.raw.Segment.Ext,
	}

	sgmnts, _ := h.SegmentFetcher.Fetch(c.Request().Context(), req.app.ID)
	sgmnt := segment.Match(sgmnts, segmentParams)
	var segmentID *int64
	if sgmnt == nil {
		segmentID = nil
	} else {
		segmentID = &sgmnt.ID
	}

	adapters, err := h.AdaptersBuilder.Build(c.Request().Context(), req.app.ID, req.adapterKeys())
	if err != nil {
		return err
	}
	if len(adapters) == 0 {
		return ErrNoAdaptersFound
	}

	resp := &ConfigResponse{
		Init: ConfigResponseInit{
			TMax:     5000,
			Adapters: adapters,
		},
		Placements: []any{},
		Token:      "{}",
		Segment:    Segment{ID: segmentID},
	}

	return c.JSON(http.StatusOK, resp)
}
