package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSettingsService_UpdatePassword(t *testing.T) {
	mockUserRepo := &UserRepoMock{
		UpdatePasswordFunc: func(ctx context.Context, userID int64, currentPassword string, newPassword string) error {
			if userID == 1 && currentPassword == "oldpassword" {
				return nil
			}
			if userID == 1 && currentPassword != "oldpassword" {
				return fmt.Errorf("current password is incorrect")
			}
			if userID != 1 {
				return fmt.Errorf("user not found")
			}
			return nil
		},
	}

	settingsService := &SettingsService{
		UserRepo: mockUserRepo,
	}

	tests := []struct {
		name                string
		authCtx             AuthContext
		requestPayload      PasswordUpdateRequest
		expectedHTTPStatus  int
		expectedErrorString string
	}{
		{
			name: "successful password update",
			authCtx: &AuthContextMock{
				UserIDFunc: func() int64 { return 1 },
				IsAdminFunc: func() bool {
					return false
				},
			},
			requestPayload: PasswordUpdateRequest{
				CurrentPassword:         "oldpassword",
				NewPassword:             "newpassword",
				NewPasswordConfirmation: "newpassword",
			},
			expectedHTTPStatus: http.StatusNoContent,
		},
		{
			name: "incorrect current password",
			authCtx: &AuthContextMock{
				UserIDFunc: func() int64 { return 1 },
				IsAdminFunc: func() bool {
					return false
				},
			},
			requestPayload: PasswordUpdateRequest{
				CurrentPassword:         "wrongpassword",
				NewPassword:             "newpassword",
				NewPasswordConfirmation: "newpassword",
			},
			expectedHTTPStatus:  http.StatusForbidden,
			expectedErrorString: "current password is incorrect",
		},
		{
			name: "user not found",
			authCtx: &AuthContextMock{
				UserIDFunc: func() int64 { return 9999 },
				IsAdminFunc: func() bool {
					return false
				},
			},
			requestPayload: PasswordUpdateRequest{
				CurrentPassword:         "oldpassword",
				NewPassword:             "newpassword",
				NewPasswordConfirmation: "newpassword",
			},
			expectedHTTPStatus:  http.StatusNotFound,
			expectedErrorString: "user not found",
		},
		{
			name: "password confirmation mismatch",
			authCtx: &AuthContextMock{
				UserIDFunc: func() int64 { return 1 },
				IsAdminFunc: func() bool {
					return false
				},
			},
			requestPayload: PasswordUpdateRequest{
				CurrentPassword:         "oldpassword",
				NewPassword:             "newpassword",
				NewPasswordConfirmation: "mismatch",
			},
			expectedHTTPStatus:  http.StatusBadRequest,
			expectedErrorString: "new_password_confirmation: New password and confirmation do not match.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			body, _ := json.Marshal(tt.requestPayload)
			req := httptest.NewRequest(http.MethodPut, "/settings/password", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			c.Set("authCtx", tt.authCtx)

			err := settingsService.UpdatePassword(c, tt.authCtx)

			if tt.expectedErrorString != "" {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				httpErr, ok := err.(*echo.HTTPError)
				if !ok {
					t.Fatalf("expected echo.HTTPError but got %T", err)
				}
				if httpErr.Code != tt.expectedHTTPStatus {
					t.Fatalf("expected HTTP status %d but got %d", tt.expectedHTTPStatus, httpErr.Code)
				}
				if httpErr.Message != tt.expectedErrorString {
					t.Fatalf("expected error message %q but got %q", tt.expectedErrorString, httpErr.Message)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if rec.Code != tt.expectedHTTPStatus {
					t.Fatalf("expected HTTP status %d but got %d", tt.expectedHTTPStatus, rec.Code)
				}
			}
		})
	}
}
