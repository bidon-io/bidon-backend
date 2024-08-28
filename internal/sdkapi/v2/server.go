package v1

import (
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/api"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/apihandlers"
	"github.com/labstack/echo/v4"
)

type Server struct {
	ConfigHandler *apihandlers.ConfigHandler
}

func (s *Server) GetConfig(c echo.Context) error {
	return s.ConfigHandler.Handle(c)
}

// Ensure that we implement the server interface
var _ api.ServerInterface = (*Server)(nil)
