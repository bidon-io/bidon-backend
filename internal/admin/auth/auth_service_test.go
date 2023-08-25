package auth_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/bidon-io/bidon-backend/config"
	"github.com/bidon-io/bidon-backend/internal/admin/auth"
	"github.com/bidon-io/bidon-backend/internal/admin/auth/mocks"
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSignUp_OK(t *testing.T) {
	userService := mocks.UserServiceMock{
		CreateUserFunc: func(email string, password string) (*db.User, error) {
			return &db.User{Email: email}, nil
		},
		ComparePasswordFunc: func(storedPasswordHash string, password string) bool {
			return true
		},
		GetUserByEmailFunc: func(email string) (*db.User, error) {
			return &db.User{Email: email}, nil
		},
	}
	tokenService := mocks.TokenServiceMock{
		GenerateAccessTokenFunc: func(email string) (string, error) {
			return "token", nil
		},
	}
	authService := auth.NewAuthService(&userService, &tokenService)

	body := `{"email":"test@test.com", "password":"pass"}`
	req := httptest.NewRequest("POST", "/auth/signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e := config.Echo("admin-test", nil)
	c := e.NewContext(req, rec)

	err := authService.SignUp(c)
	if err != nil {
		t.Fatalf("SignUp method returned an error: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Http status is not Created (201). Received: %v", rec.Code)
	}

	var response interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %s", err)
	}

	expectedResponse := map[string]any{
		"user": map[string]any{
			"email": "test@test.com",
		},
		"access_token": "token",
	}

	if !reflect.DeepEqual(response, expectedResponse) {
		t.Errorf("Response mismatch. Expected: %v. Received: %v", expectedResponse, response)
	}
}

func TestSignUp_ErrInvalidEmail(t *testing.T) {
	userService := mocks.UserServiceMock{
		CreateUserFunc: func(email string, password string) (*db.User, error) {
			return nil, errors.New("invalid email")
		},
	}
	tokenService := mocks.TokenServiceMock{
		GenerateAccessTokenFunc: func(email string) (string, error) {
			return "token", nil
		},
	}
	authService := auth.NewAuthService(&userService, &tokenService)

	body := `{"email":"test@test.com", "password":"pass"}`
	req := httptest.NewRequest("POST", "/auth/signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e := config.Echo("admin-test", nil)
	c := e.NewContext(req, rec)

	err := authService.SignUp(c)
	if he, ok := err.(*echo.HTTPError); ok {
		if !(he.Code == 422 && he.Message == "failed to create user: invalid email") {
			t.Errorf("SignUp method didn't return a correct error. Received: %v", err)
		}
	} else {
		t.Errorf("Expected error of type *echo.HTTPError, got: %T", err)
	}
}

func TestLogin_OK(t *testing.T) {
	userService := mocks.UserServiceMock{
		CreateUserFunc: func(email string, password string) (*db.User, error) {
			return &db.User{Email: email, Password: "pass"}, nil
		},
		ComparePasswordFunc: func(storedPasswordHash string, password string) bool {
			return true
		},
		GetUserByEmailFunc: func(email string) (*db.User, error) {
			return &db.User{Email: email, Password: "pass"}, nil
		},
	}
	tokenService := mocks.TokenServiceMock{
		GenerateAccessTokenFunc: func(email string) (string, error) {
			return "token", nil
		},
	}
	authService := auth.NewAuthService(&userService, &tokenService)
	body := `{"email":"test@test.com", "password":"pass"}`
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e := config.Echo("admin-test", nil)
	c := e.NewContext(req, rec)

	err := authService.LogIn(c)
	if err != nil {
		t.Fatalf("LogIn method returned an error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Http status is not ok (200). Received: %v", rec.Code)
	}

	var response interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %s", err)
	}

	expectedResponse := map[string]any{
		"user": map[string]any{
			"email": "test@test.com",
		},
		"access_token": "token",
	}

	if !reflect.DeepEqual(response, expectedResponse) {
		t.Errorf("Response mismatch. Expected: %v. Received: %v", expectedResponse, response)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	userService := mocks.UserServiceMock{
		ComparePasswordFunc: func(storedPasswordHash string, password string) bool {
			return false
		},
		GetUserByEmailFunc: func(email string) (*db.User, error) {
			return &db.User{Email: email, Password: "pass"}, nil
		},
	}
	tokenService := mocks.TokenServiceMock{
		GenerateAccessTokenFunc: func(email string) (string, error) {
			return "token", nil
		},
	}
	authService := auth.NewAuthService(&userService, &tokenService)

	body := `{"email":"test@test.com", "password":"wrongPass"}`
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e := config.Echo("admin-test", nil)
	c := e.NewContext(req, rec)

	err := authService.LogIn(c)
	if he, ok := err.(*echo.HTTPError); ok {
		if !(he.Code == 401 && he.Message == "wrong password") {
			t.Errorf("LogIn method didn't return a correct error. Received: %v", err)
		}
	} else {
		t.Errorf("Expected error of type *echo.HTTPError, got: %T", err)
	}
}
