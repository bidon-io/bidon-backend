package dspsimulator

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/prebid/openrtb/v19/openrtb2"
)

type Server struct {
	echo *echo.Echo
	svc  *Service
}

func NewServer(svc *Service) *Server {
	e := echo.New()
	s := &Server{echo: e, svc: svc}
	e.POST("/bid", s.handleBid)
	return s
}

func (s *Server) Start(addr string) error {
	return s.echo.Start(addr)
}

func (s *Server) handleBid(c echo.Context) error {
	var req openrtb2.BidRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid bid request"})
	}

	resp := s.svc.HandleBidRequest(&req)
	if resp == nil {
		return c.NoContent(http.StatusNoContent)
	}

	return c.JSON(http.StatusOK, resp)
}
