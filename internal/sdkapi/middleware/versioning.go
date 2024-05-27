package middleware

import (
	"github.com/Masterminds/semver/v3"
	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/labstack/echo/v4"
)

func Versioning(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sdkVersion, _ := semver.NewVersion(c.Request().Header.Get("X-Bidon-Version"))
		var apiVersion string

		if sdkapi.Version06GTEConstraint.Check(sdkVersion) {
			apiVersion = "v2"
		} else {
			apiVersion = "v1"
		}

		originalPath := c.Request().URL.Path
		newPath := "/" + apiVersion + originalPath

		c.SetRequest(c.Request().WithContext(c.Request().Context()))
		c.Request().URL.Path = newPath

		return next(c)
	}
}
