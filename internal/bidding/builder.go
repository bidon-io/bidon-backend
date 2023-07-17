package bidding

import (
	"context"
	"errors"
	"fmt"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/auction"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters_builder"
	"github.com/bidon-io/bidon-backend/internal/device"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"golang.org/x/exp/maps"

	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"
)

type Builder struct {
	ConfigMatcher   ConfigMatcher
	AdaptersBuilder adapters_builder.AdaptersBuilder
}

//go:generate go run -mod=mod github.com/matryer/moq@latest -out mocks_test.go . ConfigMatcher

var ErrNoBids = errors.New("no bids")

type ConfigMatcher interface {
	Match(ctx context.Context, appID int64, adType ad.Type, segmentID int64) (*auction.Config, error)
}

type BuildParams struct {
	AppID          int64
	BiddingRequest schema.BiddingRequest
	SegmentID      int64
	GeoData        geocoder.GeoData
	AdapterConfigs adapter.Config
}

func (b *Builder) HoldAuction(ctx context.Context, params *BuildParams) (adapters.DemandResponse, error) {
	// get config
	// build ortb request
	// filter adatapers
	// split to bids
	// build requests and send them to adapters in parallel
	// collect results
	// build response
	response := adapters.DemandResponse{
		Price: 0,
	}
	br := params.BiddingRequest
	config, err := b.ConfigMatcher.Match(ctx, params.AppID, br.AdType, params.SegmentID)
	if err != nil {
		return response, err
	}

	baseBidRequest := openrtb2.BidRequest{
		ID:   "123",
		Test: *bool2int(&br.Test),
		AT:   1,
		TMax: 5000,
		App: &openrtb2.App{
			Ver:    br.App.Version,
			Bundle: br.App.Bundle,
			ID:     string(params.AppID),
			Publisher: &openrtb2.Publisher{
				ID: "SELLER_ID",
			},
		},
		Device: b.BuildDevice(br.Device, br.User, params.GeoData.IPString),
		Imp:    []openrtb2.Imp{},
		Regs: &openrtb2.Regs{
			COPPA: *bool2int(&br.Regulations.COPPA),
			GDPR:  bool2int(&br.Regulations.GDPR),
		},
	}

	// Get apaters from request, demands from bidding request and demands from round config and merge them
	var roundConfig *auction.RoundConfig
	for _, round := range config.Rounds {
		if round.ID == br.Imp.RoundID {
			roundConfig = &round
			break
		}
	}
	if roundConfig == nil {
		return response, errors.New("round not found")
	}
	adapters := adapter.GetCommonAdapters(roundConfig.Bidding, br.Adapters.Keys())
	adapters = adapter.GetCommonAdapters(adapters, maps.Keys(br.Imp.Demands))

	if len(adapters) == 0 {
		return response, ErrNoBids
	}

	for _, adapterKey := range adapters {
		// adapter build bid request from baseBidRequest
		// adapter send bid request
		// adapter parse bid response
		bidder, err := b.AdaptersBuilder.Build(adapterKey, params.AdapterConfigs)
		if err != nil {
			return response, err
		}
		fmt.Println(bidder)
		fmt.Println(baseBidRequest)
		bidRequest, _ := bidder.Adapter.CreateRequest(baseBidRequest, &br)

		demandResponse := bidder.Adapter.ExecuteRequest(ctx, bidder.Client, bidRequest)
		fmt.Println(demandResponse)
		// bidder.Adapter.ParseBids(demandResponse)
	}

	return response, nil
}

func (b *Builder) BuildDevice(device schema.Device, user schema.User, ip string) *openrtb2.Device {
	return &openrtb2.Device{
		IP:             ip,
		W:              int64(device.Width),
		H:              int64(device.Height),
		JS:             int8(*device.JS),
		DeviceType:     toAdcomDeviceType(device.Type),
		ConnectionType: toAdcomConnType(device.ConnectionType),
		OS:             device.OS,
		OSV:            device.OSVersion,
		PxRatio:        device.PXRatio,
		Language:       device.Language,
		Make:           device.Manufacturer,
		HWV:            device.HardwareVersion,
		UA:             device.UserAgent,
		PPI:            int64(device.PPI),
		Model:          device.Model,
		IFA:            user.IDFA,
	}
}

func bool2int(b *bool) *int8 {
	if b == nil {
		return nil
	}
	result := int8(0)
	if *b {
		result = 1
	}
	return &result
}

func toAdcomDeviceType(deviceType device.Type) adcom1.DeviceType {
	switch deviceType {
	case device.TabletType:
		return adcom1.DeviceTablet
	case device.PhoneType:
		return adcom1.DevicePhone
	default:
		return adcom1.DeviceMobile
	}
}

func toAdcomConnType(connType string) *adcom1.ConnectionType {
	ct := adcom1.ConnectionUnknown

	switch connType {
	case "ETHERNET":
		ct = adcom1.ConnectionEthernet
	case "WIFI":
		ct = adcom1.ConnectionWIFI
	case "CELLULAR":
		ct = adcom1.ConnectionCellular
	case "CELLULAR_UNKNOWN":
		ct = adcom1.ConnectionCellular
	case "CELLULAR_2_G":
		ct = adcom1.Connection2G
	case "CELLULAR_3_G":
		ct = adcom1.Connection3G
	case "CELLULAR_4_G":
		ct = adcom1.Connection4G
	case "CELLULAR_5_G":
		ct = adcom1.Connection5G
	default:
		ct = adcom1.ConnectionUnknown
	}

	return &ct
}
