package adminecho

import (
	"github.com/bidon-io/bidon-backend/internal/admin"
	"github.com/bidon-io/bidon-backend/internal/admin/api"
	"github.com/labstack/echo/v4"
)

type Server struct {
	AppHandler              *appServiceHandler
	AppDemandProfileHandler *appDemandProfileServiceHandler
	AucCfgHandler           *auctionConfigurationServiceHandler
	AucCfgV2Handler         *auctionConfigurationV2ServiceHandler
	CountryHandler          *countryServiceHandler
	DemandSourceHandler     *demandSourceServiceHandler
}

var _ api.ServerInterface = (*Server)(nil)

func NewServer(service *admin.Service) *Server {
	appHandler := &appServiceHandler{service.AppService}
	appDemandProfileHandler := &appDemandProfileServiceHandler{service.AppDemandProfileService}
	aucHandler := &auctionConfigurationServiceHandler{service.AuctionConfigurationService}
	aucV2Handler := &auctionConfigurationV2ServiceHandler{service.AuctionConfigurationV2Service}
	countryHandler := &countryServiceHandler{service.CountryService}
	demandSourceHandler := &demandSourceServiceHandler{service.DemandSourceService}

	return &Server{
		AppHandler:              appHandler,
		AppDemandProfileHandler: appDemandProfileHandler,
		AucCfgHandler:           aucHandler,
		AucCfgV2Handler:         aucV2Handler,
		CountryHandler:          countryHandler,
		DemandSourceHandler:     demandSourceHandler,
	}
}

// App handlers

func (s *Server) GetApps(c echo.Context) error {
	return s.AppHandler.list(c)
}

func (s *Server) CreateApp(c echo.Context) error {
	return s.AppHandler.create(c)
}

func (s *Server) GetApp(c echo.Context, _ api.IdParam) error {
	return s.AppHandler.get(c)
}

func (s *Server) UpdateApp(c echo.Context, _ api.IdParam) error {
	return s.AppHandler.update(c)
}

func (s *Server) DeleteApp(c echo.Context, _ api.IdParam) error {
	return s.AppHandler.delete(c)
}

// AppDemandProfile handlers

func (s *Server) GetAppDemandProfiles(c echo.Context) error {
	return s.AppDemandProfileHandler.list(c)
}

func (s *Server) CreateAppDemandProfile(c echo.Context) error {
	return s.AppDemandProfileHandler.create(c)
}

func (s *Server) GetAppDemandProfile(c echo.Context, _ api.IdParam) error {
	return s.AppDemandProfileHandler.get(c)
}

func (s *Server) UpdateAppDemandProfile(c echo.Context, _ api.IdParam) error {
	return s.AppDemandProfileHandler.update(c)
}

func (s *Server) DeleteAppDemandProfile(c echo.Context, _ api.IdParam) error {
	return s.AppDemandProfileHandler.delete(c)
}

// AuctionConfiguration handlers

func (s *Server) GetAuctionConfigurations(c echo.Context) error {
	return s.AucCfgHandler.list(c)
}

func (s *Server) CreateAuctionConfiguration(c echo.Context) error {
	return s.AucCfgHandler.create(c)
}

func (s *Server) GetAuctionConfiguration(c echo.Context, _ api.IdParam) error {
	return s.AucCfgHandler.get(c)
}

func (s *Server) UpdateAuctionConfiguration(c echo.Context, _ api.IdParam) error {
	return s.AucCfgHandler.update(c)
}

func (s *Server) DeleteAuctionConfiguration(c echo.Context, _ api.IdParam) error {
	return s.AucCfgHandler.delete(c)
}

// AuctionConfigurationV2 handlers

func (s *Server) GetAuctionConfigurationsV2(c echo.Context) error {
	return s.AucCfgV2Handler.list(c)
}

func (s *Server) CreateAuctionConfigurationV2(c echo.Context) error {
	return s.AucCfgV2Handler.create(c)
}

func (s *Server) GetAuctionConfigurationV2(c echo.Context, _ api.IdParam) error {
	return s.AucCfgV2Handler.get(c)
}

func (s *Server) UpdateAuctionConfigurationV2(c echo.Context, _ api.IdParam) error {
	return s.AucCfgV2Handler.update(c)
}

func (s *Server) DeleteAuctionConfigurationV2(c echo.Context, _ api.IdParam) error {
	return s.AucCfgV2Handler.delete(c)
}

// Country handlers

func (s *Server) GetCountries(c echo.Context) error {
	return s.CountryHandler.list(c)
}

func (s *Server) CreateCountry(c echo.Context) error {
	return s.CountryHandler.create(c)
}

func (s *Server) GetCountry(c echo.Context, _ api.IdParam) error {
	return s.CountryHandler.get(c)
}

func (s *Server) UpdateCountry(c echo.Context, _ api.IdParam) error {
	return s.CountryHandler.update(c)
}

func (s *Server) DeleteCountry(c echo.Context, _ api.IdParam) error {
	return s.CountryHandler.delete(c)
}

// DemandSource handlers

func (s *Server) GetDemandSources(c echo.Context) error {
	return s.DemandSourceHandler.list(c)
}

func (s *Server) CreateDemandSource(c echo.Context) error {
	return s.DemandSourceHandler.create(c)
}

func (s *Server) GetDemandSource(c echo.Context, _ api.IdParam) error {
	return s.DemandSourceHandler.get(c)
}

func (s *Server) UpdateDemandSource(c echo.Context, _ api.IdParam) error {
	return s.DemandSourceHandler.update(c)
}

func (s *Server) DeleteDemandSource(c echo.Context, _ api.IdParam) error {
	return s.DemandSourceHandler.delete(c)
}
