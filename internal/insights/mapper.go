package insights

import (
	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"

	insightsopenrtb "github.com/bidon-io/bidon-backend/internal/insights/openrtb"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
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

// InitRequestFromConfigRequestWithGeoData maps sdkapi config request data and
// enriches OpenRTB geo using geocoder data.
func InitRequestFromConfigRequestWithGeoData(appID int64, req *schema.ConfigRequest, geoData geocoder.GeoData) InitRequest {
	initReq := InitRequestFromConfigRequest(appID, req)
	initReq.GeoData = geoData
	enrichOpenRTBWithGeoData(&initReq.OpenRTB, geoData)
	return initReq
}

// InitRequestFromAuctionRequest maps sdkapi auction request data to insights init payload.
func InitRequestFromAuctionRequest(appID int64, req *schema.AuctionRequest) InitRequest {
	if req == nil {
		return InitRequest{AppID: appID}
	}
	return InitRequestFromBaseRequest(appID, &req.BaseRequest)
}

// InitRequestFromAuctionRequestWithGeoData maps sdkapi auction request data and
// enriches OpenRTB geo using geocoder data.
func InitRequestFromAuctionRequestWithGeoData(appID int64, req *schema.AuctionRequest, geoData geocoder.GeoData) InitRequest {
	initReq := InitRequestFromAuctionRequest(appID, req)
	initReq.GeoData = geoData
	enrichOpenRTBWithGeoData(&initReq.OpenRTB, geoData)
	return initReq
}

// OpenRTBFromBaseRequest maps sdkapi base request data to OpenRTB-compatible insights payload.
func OpenRTBFromBaseRequest(req *schema.BaseRequest) insightsopenrtb.InitRequest {
	if req == nil {
		return insightsopenrtb.InitRequest{}
	}

	return insightsopenrtb.InitRequest{
		App:     mapOpenRTBApp(req),
		Device:  mapOpenRTBDevice(req),
		UserGeo: nil, // Use geocoder-enriched geo only.
	}
}

func mapOpenRTBApp(req *schema.BaseRequest) *openrtb2.App {
	return &openrtb2.App{
		Bundle: req.App.Bundle,
		Domain: req.App.Bundle,
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

// EnrichOpenRTBWithGeoData fills OpenRTB user.geo from geocoder data using
// the same field semantics as bidding request geo mapping.
func enrichOpenRTBWithGeoData(openRTB *insightsopenrtb.InitRequest, geoData geocoder.GeoData) {
	if openRTB == nil {
		return
	}

	if geoData.RegionCode == "" && geoData.RegionName == "" &&
		geoData.CountryCode3 == "" && geoData.CityName == "" && geoData.ZipCode == "" &&
		geoData.Lat == 0 && geoData.Lon == 0 && geoData.Accuracy == 0 {
		openRTB.UserGeo = nil
		return
	}

	openRTB.UserGeo = &openrtb2.Geo{
		Lat:       geoData.Lat,
		Lon:       geoData.Lon,
		Type:      adcom1.LocationIP,
		Accuracy:  int64(geoData.Accuracy),
		IPService: adcom1.LocationServiceMaxMind,
		Country:   geoData.CountryCode3,
		City:      geoData.CityName,
		ZIP:       geoData.ZipCode,
		Region:    geoData.RegionCode,
	}
}
