package auth

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type AuthService struct {
	userService  *UserService
	tokenService *TokenService
}

func NewAuthService(userService *UserService, tokenService *TokenService) *AuthService {
	return &AuthService{userService: userService, tokenService: tokenService}
}

func (s *AuthService) SignUp(c echo.Context) error {
	body := &AuthRequest{}
	if err := c.Bind(body); err != nil {
		return fmt.Errorf("failed to bind: %v", err)
	}

	_, err := s.userService.CreateUser(body.Email, body.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, fmt.Sprintf("failed to create user: %v", err.Error()))
	}

	token, err := s.tokenService.GenerateAccessToken(body.Email)
	if err != nil {
		return fmt.Errorf("failed generating tokens: %v", err)
	}
	publicUser := PublicUser{Email: body.Email}
	return c.JSON(http.StatusCreated, AuthResponse{User: publicUser, AccessToken: token})
}

func (s *AuthService) LogIn(c echo.Context) error {
	body := &AuthRequest{}
	if err := c.Bind(body); err != nil {
		return fmt.Errorf("failed to bind: %v", err)
	}

	user, err := s.userService.GetUserByEmail(body.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("failed to get user: %v", err))
	}
	if !s.userService.ComparePassword(user.Password, body.Password) {
		return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("wrong password"))
	}

	token, err := s.tokenService.GenerateAccessToken(body.Email)
	if err != nil {
		return fmt.Errorf("failed generating tokens: %v", err)
	}
	publicUser := PublicUser{Email: body.Email}
	return c.JSON(http.StatusOK, AuthResponse{User: publicUser, AccessToken: token})
}
