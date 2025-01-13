package admin

import (
	v8n "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"net/http"
)

type PasswordUpdateRequest struct {
	CurrentPassword         string `json:"current_password"`
	NewPassword             string `json:"new_password"`
	NewPasswordConfirmation string `json:"new_password_confirmation"`
}

func (r *PasswordUpdateRequest) Validate() error {
	return v8n.ValidateStruct(r,
		v8n.Field(&r.CurrentPassword, v8n.Required),
		v8n.Field(&r.NewPassword, v8n.Required, v8n.Length(6, 50)),
		v8n.Field(&r.NewPasswordConfirmation, v8n.Required, v8n.By(func(value interface{}) error {
			if value.(string) != r.NewPassword {
				return v8n.NewError("validation_mismatch", "New password and confirmation do not match")
			}
			return nil
		})),
	)
}

type SettingsService struct {
	UserRepo UserRepo
}

func NewSettingsService(store Store) *SettingsService {
	return &SettingsService{
		UserRepo: store.Users(),
	}
}

func (h *SettingsService) UpdatePassword(c echo.Context, authCtx AuthContext) error {
	ctx := c.Request().Context()

	var req PasswordUpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request payload").SetInternal(err)
	}

	if err := req.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if authCtx.UserID() == 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}

	err := h.UserRepo.UpdatePassword(ctx, authCtx.UserID(), req.CurrentPassword, req.NewPassword)
	if err != nil {
		if err.Error() == "current password is incorrect" {
			return echo.NewHTTPError(http.StatusForbidden, err.Error())
		}
		if err.Error() == "user not found" {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update password").SetInternal(err)
	}

	return c.NoContent(http.StatusNoContent)
}
