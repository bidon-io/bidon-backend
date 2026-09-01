package dspsim

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prebid/openrtb/v19/openrtb2"
	"go.uber.org/zap"
)

// macroPattern detects a macro the sender failed to substitute.
var macroPattern = regexp.MustCompile(`\$\{[A-Z_]+\}`)

// sizePattern pulls a size out of an asset name such as banner-320x50.svg.
var sizePattern = regexp.MustCompile(`(\d+)x(\d+)`)

// Server exposes the simulator over HTTP.
type Server struct {
	Config  Config
	Logger  *zap.Logger
	Catalog *CatalogStore
	Matcher *Matcher
	Bidder  *Bidder
	Store   *Store
}

// RegisterRoutes wires the simulator onto an Echo group.
func (s *Server) RegisterRoutes(g *echo.Group) {
	g.POST("/openrtb/bid", s.handleBid)
	g.POST("/", s.handleBid)

	for _, kind := range []string{NotificationWin, NotificationLoss, NotificationBilling} {
		path := fmt.Sprintf("/notify/%s/:bidID", kind)
		g.GET(path, s.notificationHandler(kind))
		g.POST(path, s.notificationHandler(kind))
	}

	g.GET("/creative/impression/:bidID", s.notificationHandler(NotificationImp))
	g.GET("/creative/click/:bidID", s.handleClick)
	g.GET("/creative/track/:bidID/:event", s.handleTrack)
	g.GET("/creative/asset/:name", s.handleAsset)

	g.GET("/debug/bids", s.handleDebugBids)
	g.GET("/debug/bids/:bidID", s.handleDebugBid)
	g.DELETE("/debug/bids", s.handleDebugClear)
	g.GET("/debug/creatives", s.handleDebugCreatives)
	g.GET("/debug/creatives/:dsp", s.handleDebugCreatives)
	g.GET("/debug/catalog", s.handleDebugCatalog)
	g.POST("/debug/reload", s.handleDebugReload)
}

// handleBid answers an OpenRTB bid request: 200 with a bid response, or 204
// with the reason in the X-Dspsim-Nobid-Reason header.
func (s *Server) handleBid(c echo.Context) error {
	var request openrtb2.BidRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&request); err != nil {
		s.Logger.Warn("dspsim: malformed bid request", zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("malformed bid request: %v", err))
	}

	summary, err := Describe(&request)
	if err != nil {
		return s.noBid(c, summary, ReasonNoImpression)
	}

	// Query overrides make manual testing deterministic.
	if dsp := c.QueryParam("dsp"); dsp != "" {
		summary.DSP = strings.ToLower(dsp)
	}
	adTypeOverride := ParseAdType(c.QueryParam("adtype"))
	forcedCreative := c.QueryParam("creative")

	s.Logger.Info("dspsim: bid request",
		zap.String("request_id", summary.RequestID),
		zap.String("bundle", summary.Bundle),
		zap.String("dsp", summary.DSP),
		zap.String("imp_id", summary.ImpID),
		zap.String("ad_type", AdTypeString(summary.AdType)),
		zap.String("format", string(summary.Format)),
		zap.Int64("w", summary.Width),
		zap.Int64("h", summary.Height),
		zap.Float64("floor", summary.Floor),
		zap.Bool("fullscreen", summary.Fullscreen),
	)

	match, reason := s.Matcher.Match(summary, adTypeOverride)
	if reason != "" {
		return s.noBid(c, summary, reason)
	}

	if s.Bidder.ShouldSkip() {
		return s.noBid(c, match.Summary, ReasonRandomNoBid)
	}

	response, record, reason, err := s.Bidder.Build(match, forcedCreative)
	if err != nil {
		return err
	}
	if reason != "" {
		return s.noBid(c, match.Summary, reason)
	}

	if !record.DemandConfigured {
		s.Logger.Warn("dspsim: bidding for demand that is not configured for this auction",
			zap.String("dsp", record.DSP),
			zap.String("bundle", record.Bundle),
			zap.Int64("auction_config_id", record.AuctionConfigID),
		)
	}

	s.Store.Put(record)

	if s.Config.Latency > 0 {
		time.Sleep(s.Config.Latency)
	}

	s.Logger.Info("dspsim: bid",
		zap.String("request_id", record.RequestID),
		zap.String("bid_id", record.BidID),
		zap.String("bundle", record.Bundle),
		zap.String("dsp", record.DSP),
		zap.String("ad_type", AdTypeString(record.AdType)),
		zap.Float64("floor", record.Floor),
		zap.Float64("price", record.Price),
		zap.String("creative_id", record.CreativeID),
		zap.String("creative_type", record.CreativeType),
		zap.String("creative_bucket", record.CreativeBucket),
		zap.Int64("auction_config_id", record.AuctionConfigID),
		zap.Bool("demand_configured", record.DemandConfigured),
	)

	return c.JSON(http.StatusOK, response)
}

func (s *Server) noBid(c echo.Context, summary RequestSummary, reason NoBidReason) error {
	s.Logger.Info("dspsim: no bid",
		zap.String("reason", string(reason)),
		zap.String("request_id", summary.RequestID),
		zap.String("bundle", summary.Bundle),
		zap.String("dsp", summary.DSP),
		zap.String("ad_type", AdTypeString(summary.AdType)),
		zap.String("format", string(summary.Format)),
		zap.Float64("floor", summary.Floor),
	)
	c.Response().Header().Set("X-Dspsim-Nobid-Reason", string(reason))
	return c.NoContent(http.StatusNoContent)
}

// notificationHandler records a hit on one of the URLs the simulator
// advertised. It always answers 200, as a real DSP would, even for an unknown
// bid id.
func (s *Server) notificationHandler(kind string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return s.record(c, kind, "")
	}
}

func (s *Server) handleTrack(c echo.Context) error {
	return s.record(c, NotificationTrack, c.Param("event"))
}

func (s *Server) handleClick(c echo.Context) error {
	return s.record(c, NotificationClick, "")
}

func (s *Server) record(c echo.Context, kind, event string) error {
	bidID := c.Param("bidID")
	notification := describeNotification(c, kind, event)

	known := s.Store.Record(bidID, notification)

	fields := []zap.Field{
		zap.String("kind", kind),
		zap.String("bid_id", bidID),
		zap.String("method", notification.Method),
		zap.Any("params", notification.Params),
		zap.String("remote_addr", notification.RemoteAddr),
	}
	if event != "" {
		fields = append(fields, zap.String("event", event))
	}
	if len(notification.UnresolvedMacros) > 0 {
		fields = append(fields, zap.Strings("unresolved_macros", notification.UnresolvedMacros))
	}

	switch {
	case !known:
		s.Logger.Warn("dspsim: notification for unknown bid id", fields...)
	case len(notification.UnresolvedMacros) > 0:
		s.Logger.Warn("dspsim: notification with unresolved macros", fields...)
	default:
		s.Logger.Info("dspsim: notification", fields...)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status": "ok",
		"kind":   kind,
		"bid_id": bidID,
		"known":  known,
	})
}

func describeNotification(c echo.Context, kind, event string) Notification {
	request := c.Request()
	params := map[string]string{}
	var unresolved []string

	for key, values := range c.QueryParams() {
		value := ""
		if len(values) > 0 {
			value = values[0]
		}
		params[key] = value
		if macroPattern.MatchString(value) {
			unresolved = append(unresolved, key)
		}
	}

	return Notification{
		Kind:             kind,
		Event:            event,
		ReceivedAt:       time.Now().UTC(),
		Method:           request.Method,
		Path:             request.URL.Path,
		RawQuery:         request.URL.RawQuery,
		Params:           params,
		UnresolvedMacros: unresolved,
		RemoteAddr:       c.RealIP(),
		UserAgent:        request.UserAgent(),
	}
}

// handleAsset serves creative assets. Images are generated as SVG so no binary
// files live in the repo; video assets are stubs, since nothing here needs to
// actually play.
func (s *Server) handleAsset(c echo.Context) error {
	name := c.Param("name")

	if strings.HasSuffix(name, ".mp4") {
		s.Logger.Info("dspsim: video asset requested (stub)", zap.String("name", name))
		return c.Blob(http.StatusOK, "video/mp4", nil)
	}

	w, h := int64(320), int64(50)
	if m := sizePattern.FindStringSubmatch(name); m != nil {
		if v, err := strconv.ParseInt(m[1], 10, 64); err == nil {
			w = v
		}
		if v, err := strconv.ParseInt(m[2], 10, 64); err == nil {
			h = v
		}
	}

	return c.Blob(http.StatusOK, "image/svg+xml", []byte(renderSVG(name, w, h)))
}

func renderSVG(name string, w, h int64) string {
	fontSize := h / 5
	if fontSize < 8 {
		fontSize = 8
	}
	if fontSize > 24 {
		fontSize = 24
	}
	return fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+
			`<rect width="100%%" height="100%%" fill="#1f2933"/>`+
			`<text x="50%%" y="50%%" fill="#ffffff" font-family="sans-serif" font-size="%d" `+
			`text-anchor="middle" dominant-baseline="middle">dspsim %dx%d</text>`+
			`<!-- %s --></svg>`,
		w, h, w, h, fontSize, w, h, name,
	)
}

func (s *Server) handleDebugBids(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"count":   s.Store.Len(),
		"bids":    s.Store.List(c.QueryParam("dsp")),
		"orphans": s.Store.Orphans(),
	})
}

func (s *Server) handleDebugBid(c echo.Context) error {
	record, ok := s.Store.Get(c.Param("bidID"))
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "unknown bid id")
	}
	return c.JSON(http.StatusOK, record)
}

func (s *Server) handleDebugClear(c echo.Context) error {
	s.Store.Clear()
	s.Logger.Info("dspsim: bid store cleared")
	return c.JSON(http.StatusOK, map[string]any{"status": "cleared"})
}

func (s *Server) handleDebugCreatives(c echo.Context) error {
	library := s.Bidder.GetLibrary()
	return c.JSON(http.StatusOK, map[string]any{
		"source":    library.Source,
		"buckets":   library.Buckets(),
		"creatives": library.Describe(c.Param("dsp")),
	})
}

func (s *Server) handleDebugCatalog(c echo.Context) error {
	catalog := s.Catalog.Get()
	return c.JSON(http.StatusOK, map[string]any{
		"loaded_at": catalog.LoadedAt,
		"bundles":   catalog.Bundles(),
		"auctions":  catalog.Auctions(),
	})
}

// handleDebugReload refreshes the catalog from Postgres and re-reads the
// creative library, so both can be iterated on without a restart.
func (s *Server) handleDebugReload(c echo.Context) error {
	ctx := c.Request().Context()

	result := map[string]any{}

	if err := s.Catalog.Refresh(ctx); err != nil {
		s.Logger.Error("dspsim: catalog reload failed", zap.Error(err))
		result["catalog"] = fmt.Sprintf("error: %v", err)
	} else {
		result["catalog"] = "ok"
		result["auctions"] = len(s.Catalog.Get().Auctions())
	}

	library, err := LoadLibrary(s.Config.CreativesFile)
	if err != nil {
		s.Logger.Error("dspsim: creative reload failed, keeping previous library", zap.Error(err))
		result["creatives"] = fmt.Sprintf("error: %v", err)
	} else {
		s.Bidder.SetLibrary(library)
		result["creatives"] = "ok"
		result["creatives_source"] = library.Source
	}

	return c.JSON(http.StatusOK, result)
}
