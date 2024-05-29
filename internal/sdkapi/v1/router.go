package v1

import (
	adapterstore "github.com/bidon-io/bidon-backend/internal/adapter/store"
	"github.com/bidon-io/bidon-backend/internal/auction"
	auctionstore "github.com/bidon-io/bidon-backend/internal/auction/store"
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
	GeoCoder                  *geocoder.Geocoder
	EventLogger               *event.Logger
	LineItemsMatcher          *auctionstore.LineItemsMatcher
	AdapterInitConfigsFetcher *sdkapistore.AdapterInitConfigsFetcher
	ConfigurationFetcher      *adapterstore.ConfigurationFetcher
	BiddingBuilder            *bidding.Builder
	BiddingAdaptersCfgBuilder *adapters_builder.AdaptersConfigBuilder
}

func (r *Router) RegisterRoutes(g *echo.Group) {
	auctionHandler := handlersv1.AuctionHandler{
		BaseHandler: &handlersv1.BaseHandler[schema.AuctionRequest, *schema.AuctionRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		SegmentMatcher: r.SegmentMatcher,
		AuctionBuilder: &auction.Builder{
			ConfigFetcher:    r.ConfigFetcher,
			LineItemsMatcher: r.LineItemsMatcher,
		},
		AuctionBuilderV2: &auction.BuilderV2{
			ConfigFetcher:  r.ConfigFetcher,
			AdUnitsMatcher: r.AdUnitsMatcher,
		},
		EventLogger: r.EventLogger,
	}
	configHandler := handlersv2.ConfigHandler{
		BaseHandler: &handlersv2.BaseHandler[schema.ConfigRequest, *schema.ConfigRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		SegmentMatcher:            r.SegmentMatcher,
		AdapterInitConfigsFetcher: r.AdapterInitConfigsFetcher,
		EventLogger:               r.EventLogger,
	}
	biddingHandler := handlersv1.BiddingHandler{
		BaseHandler: &handlersv1.BaseHandler[schema.BiddingRequest, *schema.BiddingRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		BiddingBuilder:        r.BiddingBuilder,
		AdaptersConfigBuilder: r.BiddingAdaptersCfgBuilder,
		AdUnitsMatcher:        r.AdUnitsMatcher,
		EventLogger:           r.EventLogger,
	}
	statsHandler := handlersv1.StatsHandler{
		BaseHandler: &handlersv1.BaseHandler[schema.StatsRequest, *schema.StatsRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		EventLogger:         r.EventLogger,
		NotificationHandler: r.NotificationHandler,
	}
	showHandler := handlersv2.ShowHandler{
		BaseHandler: &handlersv2.BaseHandler[schema.ShowRequest, *schema.ShowRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		EventLogger:         r.EventLogger,
		NotificationHandler: r.NotificationHandler,
	}
	clickHandler := handlersv2.ClickHandler{
		BaseHandler: &handlersv2.BaseHandler[schema.ClickRequest, *schema.ClickRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		EventLogger: r.EventLogger,
	}
	rewardHandler := handlersv2.RewardHandler{
		BaseHandler: &handlersv2.BaseHandler[schema.RewardRequest, *schema.RewardRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		EventLogger: r.EventLogger,
	}
	lossHandler := handlersv2.LossHandler{
		BaseHandler: &handlersv2.BaseHandler[schema.LossRequest, *schema.LossRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		EventLogger:         r.EventLogger,
		NotificationHandler: r.NotificationHandler,
	}
	winHandler := handlersv2.WinHandler{
		BaseHandler: &handlersv2.BaseHandler[schema.WinRequest, *schema.WinRequest]{
			AppFetcher:    r.AppFetcher,
			ConfigFetcher: r.ConfigFetcher,
			Geocoder:      r.GeoCoder,
		},
		EventLogger:         r.EventLogger,
		NotificationHandler: r.NotificationHandler,
	}

	g.POST("/config", configHandler.Handle)
	g.POST("/auction/:ad_type", auctionHandler.Handle)
	g.POST("/bidding/:ad_type", biddingHandler.Handle)
	g.POST("/stats/:ad_type", statsHandler.Handle)
	g.POST("/show/:ad_type", showHandler.Handle)
	g.POST("/click/:ad_type", clickHandler.Handle)
	g.POST("/reward/:ad_type", rewardHandler.Handle)
	g.POST("/loss/:ad_type", lossHandler.Handle)
	g.POST("/win/:ad_type", winHandler.Handle)

	// Legacy endpoints
	g.POST("/:ad_type/auction", auctionHandler.Handle)
	g.POST("/:ad_type/stats", statsHandler.Handle)
	g.POST("/:ad_type/show", showHandler.Handle)
	g.POST("/:ad_type/click", clickHandler.Handle)
	g.POST("/:ad_type/reward", rewardHandler.Handle)
}
