package openrtb

import "github.com/prebid/openrtb/v19/openrtb2"

// InitRequest contains OpenRTB-compatible entities passed to Insights providers.
type InitRequest struct {
	App     *openrtb2.App    `json:"app,omitempty"`
	Device  *openrtb2.Device `json:"device,omitempty"`
	UserGeo *openrtb2.Geo    `json:"user_geo,omitempty"`
}
