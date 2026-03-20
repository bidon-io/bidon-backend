package insights

import (
	"github.com/prebid/openrtb/v19/openrtb2"

	insightsopenrtb "github.com/bidon-io/bidon-backend/internal/insights/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

// InitRequestFromBaseRequest maps common sdkapi base request data to insights init payload.
func InitRequestFromBaseRequest(appID int64, req *schema.BaseRequest) InitRequest {
	if req == nil {
		return InitRequest{AppID: appID}
	}
	baseRequest := *req

	return InitRequest{
		AppID:       appID,
		BaseRequest: &baseRequest,
		AppVersion:  req.App.Version,
		SDKVersion:  req.App.SDKVersion,
		IDFA:        req.User.IDFA,
		IDG:         req.User.IDG,
		IDFV:        req.User.IDFV,
		OpenRTB:     OpenRTBFromBaseRequest(req),
	}
}

// InitRequestFromConfigRequest maps sdkapi config request data to insights init payload.
func InitRequestFromConfigRequest(appID int64, req *schema.ConfigRequest) InitRequest {
	if req == nil {
		return InitRequest{AppID: appID}
	}
	return InitRequestFromBaseRequest(appID, &req.BaseRequest)
}

// InitRequestFromAuctionRequest maps sdkapi auction request data to insights init payload.
func InitRequestFromAuctionRequest(appID int64, req *schema.AuctionRequest) InitRequest {
	if req == nil {
		return InitRequest{AppID: appID}
	}
	return InitRequestFromBaseRequest(appID, &req.BaseRequest)
}

// OpenRTBFromBaseRequest maps sdkapi base request data to OpenRTB-compatible insights payload.
func OpenRTBFromBaseRequest(req *schema.BaseRequest) insightsopenrtb.InitRequest {
	if req == nil {
		return insightsopenrtb.InitRequest{}
	}

	return insightsopenrtb.InitRequest{
		App:     mapOpenRTBApp(req),
		Device:  mapOpenRTBDevice(req),
		UserGeo: mapOpenRTBUserGeo(req),
	}
}

func mapOpenRTBApp(req *schema.BaseRequest) *openrtb2.App {
	return &openrtb2.App{
		Bundle: req.App.Bundle,
		Ver:    req.App.Version,
	}
}

func mapOpenRTBDevice(req *schema.BaseRequest) *openrtb2.Device {
	d := &openrtb2.Device{
		UA:       req.Device.UserAgent,
		Make:     req.Device.Manufacturer,
		Model:    req.Device.Model,
		OS:       req.Device.OS,
		OSV:      req.Device.OSVersion,
		HWV:      req.Device.HardwareVersion,
		Language: req.Device.Language,
		IP:       req.Device.IP,
		Carrier:  req.Device.Carrier,
		MCCMNC:   req.Device.MCCMNC,
		W:        int64(req.Device.Width),
		H:        int64(req.Device.Height),
		PPI:      int64(req.Device.PPI),
		PxRatio:  req.Device.PXRatio,
		IFA:      req.User.IDFA,
	}

	if req.Device.JS != nil {
		d.JS = int8(*req.Device.JS)
	}

	return d
}

func mapOpenRTBUserGeo(req *schema.BaseRequest) *openrtb2.Geo {
	geo := req.GetGeo()
	if isEmptyGeo(geo) {
		return nil
	}

	return &openrtb2.Geo{
		Lat:      geo.Lat,
		Lon:      geo.Lon,
		Accuracy: int64(geo.Accuracy),
		Country:  geo.Country,
		City:     geo.City,
		ZIP:      geo.ZIP,
	}
}

func isEmptyGeo(geo schema.Geo) bool {
	return geo.Lat == 0 &&
		geo.Lon == 0 &&
		geo.Accuracy == 0 &&
		geo.Country == "" &&
		geo.City == "" &&
		geo.ZIP == ""
}
