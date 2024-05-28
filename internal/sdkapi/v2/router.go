package v1

import (
	adapterstore "github.com/bidon-io/bidon-backend/internal/adapter/store"
	auctionstore "github.com/bidon-io/bidon-backend/internal/auction/store"
	"github.com/bidon-io/bidon-backend/internal/auctionv2"
	"github.com/bidon-io/bidon-backend/internal/bidding"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters_builder"
	"github.com/bidon-io/bidon-backend/internal/notification"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/geocoder"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	sdkapistore "github.com/bidon-io/bidon-backend/internal/sdkapi/store"
	handlersv1 "github.com/bidon-io/bidon-backend/internal/sdkapi/v1/handlers"
	handlersv2 "github.com/bidon-io/bidon-backend/internal/sdkapi/v2/handlers"
	"github.com/bidon-io/bidon-backend/internal/segment"
	"github.com/labstack/echo/v4"
)

type Router struct {
	ConfigFetcher             *auctionstore.ConfigFetcher
	AppFetcher                *sdkapistore.AppFetcher
	SegmentMatcher            *segment.Matcher
	AdUnitsMatcher            *auctionstore.AdUnitsMatcher
	NotificationHandler       notification.Handler
	NotificationHandlerV2     notification.HandlerV2
	GeoCoder                  *geocoder.Geocoder
	EventLogger               *event.Logger
	LineItemsMatcher          *auctionstore.LineItemsMatcher
	AdapterInitConfigsFetcher *sdkapistore.AdapterInitConfigsFetcher
	ConfigurationFetcher      *adapterstore.ConfigurationFetcher
	BiddingBuilder            *bidding.Builder
	BiddingAdaptersCfgBuilder *adapters_builder.AdaptersConfigBuilder
}

func (r *Router) RegisterRoutes(g *echo.Group) {
	auctionHandler := handlersv2.AuctionHandler{
		BaseHandler: &handlersv2.BaseHandler[schema.AuctionRequest, *schema.AuctionRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		SegmentMatcher: r.SegmentMatcher,
		AuctionBuilder: &auctionv2.Builder{
			ConfigFetcher:                r.ConfigFetcher,
			AdUnitsMatcher:               r.AdUnitsMatcher,
			BiddingBuilder:               r.BiddingBuilder,
			BiddingAdaptersConfigBuilder: r.BiddingAdaptersCfgBuilder,
		},
		EventLogger: r.EventLogger,
	}
	statsHandler := handlersv2.StatsHandler{
		BaseHandler: &handlersv2.BaseHandler[schema.StatsRequest, *schema.StatsRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		EventLogger:         r.EventLogger,
		NotificationHandler: r.NotificationHandler,
	}
	configHandler := handlersv1.ConfigHandler{
		BaseHandler: &handlersv1.BaseHandler[schema.ConfigRequest, *schema.ConfigRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		SegmentMatcher:            r.SegmentMatcher,
		AdapterInitConfigsFetcher: r.AdapterInitConfigsFetcher,
		EventLogger:               r.EventLogger,
	}
	showHandler := handlersv1.ShowHandler{
		BaseHandler: &handlersv1.BaseHandler[schema.ShowRequest, *schema.ShowRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		EventLogger:         r.EventLogger,
		NotificationHandler: r.NotificationHandler,
	}
	clickHandler := handlersv1.ClickHandler{
		BaseHandler: &handlersv1.BaseHandler[schema.ClickRequest, *schema.ClickRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		EventLogger: r.EventLogger,
	}
	rewardHandler := handlersv1.RewardHandler{
		BaseHandler: &handlersv1.BaseHandler[schema.RewardRequest, *schema.RewardRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		EventLogger: r.EventLogger,
	}
	lossHandler := handlersv1.LossHandler{
		BaseHandler: &handlersv1.BaseHandler[schema.LossRequest, *schema.LossRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		EventLogger:         r.EventLogger,
		NotificationHandler: r.NotificationHandler,
	}
	winHandler := handlersv1.WinHandler{
		BaseHandler: &handlersv1.BaseHandler[schema.WinRequest, *schema.WinRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		EventLogger:         r.EventLogger,
		NotificationHandler: r.NotificationHandler,
	}

	g.POST("/config", configHandler.Handle)
	g.POST("/auction/:ad_type", auctionHandler.Handle)
	g.POST("/stats/:ad_type", statsHandler.Handle)
	g.POST("/show/:ad_type", showHandler.Handle)
	g.POST("/click/:ad_type", clickHandler.Handle)
	g.POST("/reward/:ad_type", rewardHandler.Handle)
	g.POST("/loss/:ad_type", lossHandler.Handle)
	g.POST("/win/:ad_type", winHandler.Handle)
}
