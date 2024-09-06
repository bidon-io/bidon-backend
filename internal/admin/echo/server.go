package adminecho

import (
	"github.com/bidon-io/bidon-backend/internal/admin"
	"github.com/bidon-io/bidon-backend/internal/admin/api"
	"github.com/labstack/echo/v4"
)

type Server struct {
	AppHandler                 *appServiceHandler
	AppDemandProfileHandler    *appDemandProfileServiceHandler
	AucCfgHandler              *auctionConfigurationServiceHandler
	AucCfgV2Handler            *auctionConfigurationV2ServiceHandler
	CountryHandler             *countryServiceHandler
	DemandSourceHandler        *demandSourceServiceHandler
	DemandSourceAccountHandler *demandSourceAccountServiceHandler
	LineItemHandler            *lineItemServiceHandler
	SegmentHandler             *segmentServiceHandler
	UserHandler                *userHandler
}

var _ api.ServerInterface = (*Server)(nil)

func NewServer(service *admin.Service) *Server {
	appHandler := &appServiceHandler{service.AppService}
	appDemandProfileHandler := &appDemandProfileServiceHandler{service.AppDemandProfileService}
	aucHandler := &auctionConfigurationServiceHandler{service.AuctionConfigurationService}
	aucV2Handler := &auctionConfigurationV2ServiceHandler{service.AuctionConfigurationV2Service}
	countryHandler := &countryServiceHandler{service.CountryService}
	demandSourceHandler := &demandSourceServiceHandler{service.DemandSourceService}
	demandSourceAccountHandler := &demandSourceAccountServiceHandler{service.DemandSourceAccountService}
	lineItemHandler := &lineItemServiceHandler{service.LineItemService}
	segmentHandler := &segmentServiceHandler{service.SegmentService}
	usrHandler := &userHandler{
		userServiceHandler: &userServiceHandler{service.UserService},
	}

	return &Server{
		AppHandler:                 appHandler,
		AppDemandProfileHandler:    appDemandProfileHandler,
		AucCfgHandler:              aucHandler,
		AucCfgV2Handler:            aucV2Handler,
		CountryHandler:             countryHandler,
		DemandSourceHandler:        demandSourceHandler,
		DemandSourceAccountHandler: demandSourceAccountHandler,
		LineItemHandler:            lineItemHandler,
		SegmentHandler:             segmentHandler,
		UserHandler:                usrHandler,
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

// Demand Source Account handlers

func (s *Server) GetDemandSourceAccounts(c echo.Context) error {
	return s.DemandSourceAccountHandler.list(c)
}

func (s *Server) CreateDemandSourceAccount(c echo.Context) error {
	return s.DemandSourceAccountHandler.create(c)
}

func (s *Server) GetDemandSourceAccount(c echo.Context, _ api.IdParam) error {
	return s.DemandSourceAccountHandler.get(c)
}

func (s *Server) UpdateDemandSourceAccount(c echo.Context, _ api.IdParam) error {
	return s.DemandSourceAccountHandler.update(c)
}

func (s *Server) DeleteDemandSourceAccount(c echo.Context, _ api.IdParam) error {
	return s.DemandSourceAccountHandler.delete(c)
}

// Segment handlers

func (s *Server) GetSegments(c echo.Context) error {
	return s.SegmentHandler.list(c)
}

func (s *Server) CreateSegment(c echo.Context) error {
	return s.SegmentHandler.create(c)
}

func (s *Server) GetSegment(c echo.Context, _ api.IdParam) error {
	return s.SegmentHandler.get(c)
}

func (s *Server) UpdateSegment(c echo.Context, _ api.IdParam) error {
	return s.SegmentHandler.update(c)
}

func (s *Server) DeleteSegment(c echo.Context, _ api.IdParam) error {
	return s.SegmentHandler.delete(c)
}

// User handlers

func (s *Server) GetUsers(c echo.Context) error {
	return s.UserHandler.list(c)
}

func (s *Server) CreateUser(c echo.Context) error {
	return s.UserHandler.create(c)
}

func (s *Server) GetUser(c echo.Context, _ api.IdParam) error {
	return s.UserHandler.get(c)
}

func (s *Server) GetCurrentUser(c echo.Context) error {
	return s.UserHandler.get(c)
}

func (s *Server) UpdateUser(c echo.Context, _ api.IdParam) error {
	return s.UserHandler.update(c)
}

func (s *Server) DeleteUser(c echo.Context, _ api.IdParam) error {
	return s.UserHandler.delete(c)
}

// LineItem handlers

func (s *Server) GetLineItems(c echo.Context, _ api.GetLineItemsParams) error {
	return s.LineItemHandler.list(c)
}

func (s *Server) CreateLineItem(c echo.Context) error {
	return s.LineItemHandler.create(c)
}

func (s *Server) GetLineItem(c echo.Context, _ api.IdParam) error {
	return s.LineItemHandler.get(c)
}

func (s *Server) UpdateLineItem(c echo.Context, _ api.IdParam) error {
	return s.LineItemHandler.update(c)
}

func (s *Server) DeleteLineItem(c echo.Context, _ api.IdParam) error {
	return s.LineItemHandler.delete(c)
}
