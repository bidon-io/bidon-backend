package v1

import (
	"encoding/json"
	"fmt"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/api"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/v2/apihandlers"
	"github.com/labstack/echo/v4"
	"net/http"
)

type Server struct {
	ConfigHandler *apihandlers.ConfigHandler
}

func (s *Server) GetConfig(c echo.Context) error {
	return s.ConfigHandler.Handle(c)
}

func (s *Server) GetOpenAPISpec(c echo.Context) error {
	spec, err := api.GetSwagger()
	if err != nil {
		return err
	}

	swaggerJSON, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to generate OpenAPI spec")
	}

	return c.JSONBlob(http.StatusOK, swaggerJSON)
}

// Ensure that we implement the server interface
var _ api.ServerInterface = (*Server)(nil)
