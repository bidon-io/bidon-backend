package grpcserver

import (
	"context"
	"log"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/auction"
	"github.com/bidon-io/bidon-backend/internal/auctionv2"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	v3 "github.com/bidon-io/bidon-backend/pkg/proto/com/iabtechlab/openrtb/v3"
	pb "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/proto/v1"
)

type Server struct {
	pb.UnimplementedBiddingServiceServer
	AuctionService AuctionService
	AppFetcher     AppFetcher
}

func NewServer(auctionService AuctionService, appFetcher AppFetcher) *Server {
	return &Server{
		AuctionService: auctionService,
		AppFetcher:     appFetcher,
	}
}

//go:generate go run -mod=mod github.com/matryer/moq@latest -out mocks/mocks.go -pkg mocks . AppFetcher AuctionService

type AppFetcher interface {
	FetchCached(ctx context.Context, appKey, appBundle string) (sdkapi.App, error)
}

type ConfigFetcher interface {
	FetchByUIDCached(ctx context.Context, appId int64, id, uid string) *auction.Config
	Match(ctx context.Context, appID int64, adType ad.Type, segmentID int64, version string) (*auction.Config, error)
}

type AuctionService interface {
	Run(ctx context.Context, params *auctionv2.ExecutionParams) (*auctionv2.Response, error)
}

func (s *Server) Bid(ctx context.Context, o *v3.Openrtb) (*v3.Openrtb, error) {
	adapter := NewAuctionAdapter()
	ar, err := adapter.OpenRTBToAuctionRequest(o)
	if err != nil {
		return &v3.Openrtb{}, err
	}

	app, err := s.AppFetcher.FetchCached(ctx, ar.App.Key, ar.App.Bundle)
	if err != nil {
		return &v3.Openrtb{}, err
	}

	params := &auctionv2.ExecutionParams{
		Req:     ar,
		AppID:   app.ID,
		Country: ar.Geo.Country,
		GeoData: buildGeoData(ar.Geo),
		Log: func(s string) {
			log.Print(s)
		},
		LogErr: func(err error) {
			log.Print(err)
		},
	}

	result, err := s.AuctionService.Run(ctx, params)
	if err != nil {
		return &v3.Openrtb{}, err
	}

	response, err := adapter.AuctionResponseToOpenRTB(result)
	if err != nil {
		return &v3.Openrtb{}, err
	}

	return response, nil
}

// TODO: we don't have IP - which is necessary for bidding
func buildGeoData(geo *schema.Geo) geocoder.GeoData {
	return geocoder.GeoData{
		CountryCode: geo.Country,
		CityName:    geo.City,
		Lat:         geo.Lat,
		Lon:         geo.Lon,
		Accuracy:    int(geo.Accuracy),
	}
}
