package sdkapi

import (
	"fmt"
	"net/http"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/event"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/labstack/echo/v4"
)

type WinHandler struct {
	*BaseHandler[schema.WinRequest, *schema.WinRequest]
	EventLogger         *event.Logger
	NotificationHandler NotificationHandler
}

func (h *WinHandler) Handle(c echo.Context) error {
	req, err := h.resolveRequest(c)
	if err != nil {
		return err
	}

	winEvent := event.NewWin(&req.raw, req.geoData)
	h.EventLogger.Log(winEvent, func(err error) {
		logError(c, fmt.Errorf("log win event: %v", err))
	})

	return c.JSON(http.StatusOK, map[string]any{"success": true})
}
