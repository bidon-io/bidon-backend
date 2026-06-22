// Package adminecho implements Echo bindings for the admin package.
package adminecho

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	v8n "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	session "github.com/spazzymoto/echo-scs-session"

	"github.com/bidon-io/bidon-backend/internal/admin"
	"github.com/bidon-io/bidon-backend/internal/admin/auth"
	"github.com/bidon-io/bidon-backend/internal/admin/resource"
	"github.com/bidon-io/bidon-backend/internal/audit"
)

func UseAuthorization(g *echo.Group, authService *auth.Service) {
	sm := authService.GetSessionManager()
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			skipper := skipIfAny(skipIfWebAppOrAuth("Bearer", "Basic"), skipIfAuthRoutes())
			if skipper(c) {
				return next(c)
			}

			apiKey := c.Request().Header.Get("X-Bidon-Api-Key")
			if apiKey == "" {
				return next(c)
			}

			authCtx, err := authService.ResolveAPIKey(c.Request().Context(), apiKey)
			if err != nil {
				return err
			}

			c.Set("authCtx", authCtx)
			c.Set("authMethod", audit.AuthAPIKey)
			if apiKeyCtx, ok := authCtx.(*auth.APIKey); ok {
				c.Set("apiKeyID", apiKeyCtx.ID.String())
			}

			return next(c)
		}
	})
	g.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
		Skipper: skipIfAny(skipIfWebAppOrAuth("Bearer"), skipIfAuthRoutes(), skipIfApiKey(), skipIfOpenAPISpec()),
		Validator: func(username, password string, c echo.Context) (bool, error) {
			if authService.IsSuperUser(username, password) {
				c.Set("authCtx", stubAuthContext{})
				c.Set("authMethod", audit.AuthBasic)

				return true, nil
			}

			return false, nil
		},
	}))
	g.Use(echojwt.WithConfig(echojwt.Config{
		Skipper: skipIfAny(skipIfWebAppOrAuth("Basic"), skipIfAuthRoutes(), skipIfApiKey(), skipIfOpenAPISpec()),
		SuccessHandler: func(c echo.Context) {
			token := c.Get("user").(*jwt.Token)
			claims := token.Claims.(*auth.JWTClaims)

			c.Set("authCtx", claims)
			c.Set("authMethod", audit.AuthSession)
		},
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(auth.JWTClaims)
		},
		KeyFunc: func(_ *jwt.Token) (any, error) {
			return authService.GetSecretKey(), nil
		},
	}))
	g.Use(session.LoadAndSaveWithConfig(session.SessionConfig{
		Skipper:        skipIfAny(skipIfNotWebApp(), skipIfAuthRoutes(), skipIfOpenAPISpec()),
		SessionManager: sm,
	}))
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			skipper := skipIfAny(skipIfNotWebApp(), skipIfAuthRoutes(), skipIfOpenAPISpec())
			if skipper(c) {
				return next(c)
			}

			authCtx := authService.NewSessionAuthContext(c.Request().Context())
			if authCtx != nil {
				c.Set("authCtx", authCtx)
				c.Set("authMethod", audit.AuthSession)
			}

			return next(c)
		}
	})

	// Inject audit context into request context
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			if authCtx, ok := c.Get("authCtx").(admin.AuthContext); ok {
				ctx = audit.WithUserID(ctx, authCtx.UserID())
			}
			if authMethod, ok := c.Get("authMethod").(string); ok {
				ctx = audit.WithAuthMethod(ctx, authMethod)
			}
			if apiKeyID, ok := c.Get("apiKeyID").(string); ok {
				ctx = audit.WithAPIKeyID(ctx, apiKeyID)
			}
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	})
}

type appServiceHandler = resourceServiceHandler[admin.AppResource, admin.AppAttrs]
type appDemandProfileServiceHandler = resourceServiceHandler[admin.AppDemandProfileResource, admin.AppDemandProfileAttrs]
type auctionConfigurationServiceHandler = resourceServiceHandler[admin.AuctionConfigurationResource, admin.AuctionConfigurationAttrs]
type auctionConfigurationV2ServiceHandler = resourceServiceHandler[admin.AuctionConfigurationV2Resource, admin.AuctionConfigurationV2Attrs]
type countryServiceHandler = resourceServiceHandler[admin.CountryResource, admin.CountryAttrs]
type demandSourceServiceHandler = resourceServiceHandler[admin.DemandSourceResource, admin.DemandSourceAttrs]
type demandSourceAccountServiceHandler = resourceServiceHandler[admin.DemandSourceAccountResource, admin.DemandSourceAccountAttrs]
type lineItemServiceHandler = resourceServiceHandler[admin.LineItemResource, admin.LineItemAttrs]
type segmentServiceHandler = resourceServiceHandler[admin.SegmentResource, admin.SegmentAttrs]
type userServiceHandler = resourceServiceHandler[admin.UserResource, admin.UserAttrs]
type settingsServiceHandler struct {
	service *admin.SettingsService
}

type resourceServiceHandler[Resource, ResourceAttrs any] struct {
	service resourceService[Resource, ResourceAttrs]
}

type resourceService[Resource, ResourceAttrs any] interface {
	List(ctx context.Context, authCtx admin.AuthContext, qParams map[string][]string) (*resource.Collection[Resource], error)
	Find(ctx context.Context, authCtx admin.AuthContext, id int64) (*Resource, error)
	Create(ctx context.Context, authCtx admin.AuthContext, attrs *ResourceAttrs) (*Resource, error)
	Update(ctx context.Context, authCtx admin.AuthContext, id int64, attrs *ResourceAttrs) (*Resource, error)
	Delete(ctx context.Context, authCtx admin.AuthContext, id int64) error
}

// stubAuthContext is a stub implementation of admin.AuthContext. Build auth context from JWT token.
type stubAuthContext struct{}

func (s stubAuthContext) UserID() int64 {
	return 0
}

func (s stubAuthContext) IsAdmin() bool {
	return true
}

func (s *resourceServiceHandler[Resource, ResourceAttrs]) list(c echo.Context) error {
	authCtx, err := getAuthContext(c)
	if err != nil {
		return err
	}

	collection, err := s.service.List(c.Request().Context(), authCtx, c.QueryParams())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, collection.Items)
}

func (s *resourceServiceHandler[Resource, ResourceAttrs]) listCollection(c echo.Context) error {
	authCtx, err := getAuthContext(c)
	if err != nil {
		return err
	}

	collection, err := s.service.List(c.Request().Context(), authCtx, c.QueryParams())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, collection)
}

func (s *resourceServiceHandler[Resource, ResourceAttrs]) create(c echo.Context) error {
	return s.createWithStatus(c, nil)
}

func (s *resourceServiceHandler[Resource, ResourceAttrs]) createWithStatus(
	c echo.Context,
	statusCode func(resource *Resource) int,
) error {
	authCtx, err := getAuthContext(c)
	if err != nil {
		return err
	}

	attrs := new(ResourceAttrs)
	if err := c.Bind(attrs); err != nil {
		return err
	}

	resource, err := s.service.Create(c.Request().Context(), authCtx, attrs)
	if err != nil {
		var validationError v8n.Errors
		if errors.As(err, &validationError) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, validationError.Error())
		}

		return err
	}

	status := http.StatusCreated
	if statusCode != nil {
		if customStatus := statusCode(resource); customStatus != 0 {
			status = customStatus
		}
	}
	return c.JSON(status, resource)
}

func (s *resourceServiceHandler[Resource, ResourceAttrs]) get(c echo.Context) error {
	authCtx, err := getAuthContext(c)
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid id: %v", err)
	}

	resource, err := s.service.Find(c.Request().Context(), authCtx, int64(id))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, resource)
}

func (s *resourceServiceHandler[Resource, ResourceAttrs]) update(c echo.Context) error {
	authCtx, err := getAuthContext(c)
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid id: %v", err)
	}

	attrs := new(ResourceAttrs)
	if err := c.Bind(attrs); err != nil {
		return err
	}

	resource, err := s.service.Update(c.Request().Context(), authCtx, int64(id), attrs)
	if err != nil {
		var validationError v8n.Errors
		if errors.As(err, &validationError) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, validationError.Error())
		}

		return err
	}

	return c.JSON(http.StatusOK, resource)
}

func (s *resourceServiceHandler[Resource, ResourceAttrs]) delete(c echo.Context) error {
	authCtx, err := getAuthContext(c)
	if err != nil {
		return err
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid id: %v", err)
	}

	if err := s.service.Delete(c.Request().Context(), authCtx, int64(id)); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *settingsServiceHandler) updatePassword(c echo.Context, authCtx admin.AuthContext) error {
	return h.service.UpdatePassword(c, authCtx)
}

func getAuthContext(c echo.Context) (admin.AuthContext, error) {
	authCtx, ok := c.Get("authCtx").(admin.AuthContext)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "unauthorized").SetInternal(
			fmt.Errorf("failed to get auth context from request"),
		)
	}

	return authCtx, nil
}

func skipIfApiKey() middleware.Skipper {
	return func(c echo.Context) bool {
		return c.Request().Header.Get("X-Bidon-Api-Key") != ""
	}
}

func skipIfWebAppOrAuth(prefixes ...string) middleware.Skipper {
	webAppSkipper := skipIfWebApp()
	authSkipper := skipIfAuthIs(prefixes...)
	return func(c echo.Context) bool {
		return webAppSkipper(c) || authSkipper(c)
	}
}

func skipIfWebApp() middleware.Skipper {
	return func(c echo.Context) bool {
		return c.Request().Header.Get("X-Bidon-App") == "web"
	}
}

func skipIfNotWebApp() middleware.Skipper {
	return func(c echo.Context) bool {
		return c.Request().Header.Get("X-Bidon-App") != "web"
	}
}

func skipIfAuthIs(prefixes ...string) middleware.Skipper {
	return func(c echo.Context) bool {
		header := c.Request().Header.Get(echo.HeaderAuthorization)

		for _, prefix := range prefixes {
			prefix = prefix + " "
			if len(header) >= len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
				return true
			}
		}

		return false
	}
}

func skipIfAuthRoutes() middleware.Skipper {
	return func(c echo.Context) bool {
		return strings.HasPrefix(c.Path(), "/auth")
	}
}

func skipIfOpenAPISpec() middleware.Skipper {
	return func(c echo.Context) bool {
		return c.Path() == "/api/openapi.json"
	}
}

// Combine skippers with OR logic
func skipIfAny(skippers ...middleware.Skipper) middleware.Skipper {
	return func(c echo.Context) bool {
		// Any skipper returning true will make the combined skipper skip
		for _, skipper := range skippers {
			if skipper(c) {
				return true
			}
		}
		return false
	}
}
