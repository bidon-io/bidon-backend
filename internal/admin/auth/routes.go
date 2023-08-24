package auth

import (
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

func SetUpRoutes(e *echo.Echo, db *db.DB, jwtSecretKey []byte) {
	userService := NewUserService(db)
	tokenService := NewTokenService(jwtSecretKey)
	authService := NewAuthService(userService, tokenService)

	e.POST("/auth/signup", authService.SignUp)
	e.POST("/auth/login", authService.LogIn)
}

func ConfigureJWT(g *echo.Group, jwtSecretKey []byte) {
	config := echojwt.Config{
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(JwtCustomClaims)
		},
		SigningKey: jwtSecretKey,
	}
	g.Use(echojwt.WithConfig(config))
}
