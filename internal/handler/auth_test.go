package handler

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal/security"
	"github.com/haatos/simple-ci/internal/service"
	"github.com/haatos/simple-ci/internal/settings"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/testutil"
	"github.com/haatos/simple-ci/internal/util"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthHandler_GetLoginPage(t *testing.T) {
	t.Run("success - login page html is returned", func(t *testing.T) {
		// arrange
		mockService := new(testutil.MockUserService)
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewAuthHandler(mockService, nil)

		// act
		err := h.GetLoginPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<html")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `name="username"`)
		assert.Contains(t, body, `name="password"`)
	})
	t.Run("success - login page main html is returned for htmx request", func(t *testing.T) {
		// arrange
		mockService := new(testutil.MockUserService)
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		c.Request().Header.Set("hx-request", "true")
		h := NewAuthHandler(mockService, nil)

		// act
		err := h.GetLoginPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.NotContains(t, body, "<html")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `name="username"`)
		assert.Contains(t, body, `name="password"`)
	})
	t.Run("success - logged in user is redirected to app page", func(t *testing.T) {
		// arrange
		mockService := new(testutil.MockUserService)
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		user := &store.User{
			UserID:       1,
			UserRoleID:   store.Operator,
			Username:     "username",
			PasswordHash: "passwordhash",
		}
		c.Set("user", user)
		c.Request().Header.Set("hx-request", "true")
		h := NewAuthHandler(mockService, nil)

		// act
		err := h.GetLoginPage(c)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/app", rec.Header().Get(echo.HeaderLocation))
	})
}

func TestAuthHandler_PostLogin(t *testing.T) {
	t.Run("success - user logs in", func(t *testing.T) {
		// arrange
		settings.Settings = settings.NewSettings()
		mockService := new(testutil.MockUserService)
		expectedUser := &store.User{
			UserID:            1,
			UserRoleID:        store.Operator,
			Username:          "testuser",
			PasswordHash:      "testuserpasswordhash",
			PasswordChangedOn: util.AsPtr(time.Now().UTC()),
		}
		expectedSession := &store.AuthSession{
			AuthSessionID:      "thisisasessionid",
			AuthSessionUserID:  expectedUser.UserID,
			AuthSessionExpires: time.Now().UTC().Add(30 * time.Second),
		}
		mockService.On(
			"GetUserByUsernameAndPassword",
			context.Background(),
			expectedUser.Username,
			"password",
		).Return(expectedUser, nil)
		mockService.On(
			"CreateAuthSession",
			context.Background(),
			expectedSession.AuthSessionUserID,
		).Return(expectedSession, nil)

		e := echo.New()
		formData := url.Values{}
		formData.Set("username", expectedUser.Username)
		formData.Set("password", "password")
		req := httptest.NewRequest(
			http.MethodPost,
			"/auth/login",
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		h := NewAuthHandler(
			mockService,
			service.NewCookieService(
				[]byte(security.GenerateRandomKey(32)),
				[]byte(security.GenerateRandomKey(24)),
			),
		)

		// act
		err := h.PostLogin(c)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, "/app", rec.Header().Get("hx-redirect"))
	})
	t.Run("failure - invalid username", func(t *testing.T) {
		// arrange
		settings.Settings = settings.NewSettings()
		mockService := new(testutil.MockUserService)
		mockService.On(
			"GetUserByUsernameAndPassword",
			context.Background(),
			"testuser",
			"password",
		).Return(nil, sql.ErrNoRows)

		e := echo.New()
		formData := url.Values{}
		formData.Set("username", "testuser")
		formData.Set("password", "password")
		req := httptest.NewRequest(
			http.MethodPost,
			"/auth/login",
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		h := NewAuthHandler(
			mockService,
			service.NewCookieService(
				[]byte(security.GenerateRandomKey(32)),
				[]byte(security.GenerateRandomKey(24)),
			),
		)

		// act
		err := h.PostLogin(c)

		// assert
		assert.Error(t, err)
		htmxErr, ok := err.(*HTMXError)
		assert.True(t, ok)
		assert.Contains(t, htmxErr.Message, "invalid username")
	})
	t.Run("failure - invalid password", func(t *testing.T) {
		// arrange
		settings.Settings = settings.NewSettings()
		mockService := new(testutil.MockUserService)
		mockService.On(
			"GetUserByUsernameAndPassword",
			context.Background(),
			"testuser",
			"password",
		).Return(nil, bcrypt.ErrMismatchedHashAndPassword)

		e := echo.New()
		formData := url.Values{}
		formData.Set("username", "testuser")
		formData.Set("password", "password")
		req := httptest.NewRequest(
			http.MethodPost,
			"/auth/login",
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		h := NewAuthHandler(
			mockService,
			service.NewCookieService(
				[]byte(security.GenerateRandomKey(32)),
				[]byte(security.GenerateRandomKey(24)),
			),
		)

		// act
		err := h.PostLogin(c)

		// assert
		assert.Error(t, err)
		htmxErr, ok := err.(*HTMXError)
		assert.True(t, ok)
		assert.Equal(t, "invalid username or password", htmxErr.Message)
	})
	t.Run("success - new user is redirected to change password", func(t *testing.T) {
		// arrange
		settings.Settings = settings.NewSettings()
		mockService := new(testutil.MockUserService)
		expectedUser := &store.User{
			UserID:       1,
			UserRoleID:   store.Operator,
			Username:     "testuser",
			PasswordHash: "testuserpasswordhash",
		}
		expectedSession := &store.AuthSession{
			AuthSessionID:      "thisisasessionid",
			AuthSessionUserID:  expectedUser.UserID,
			AuthSessionExpires: time.Now().UTC().Add(30 * time.Second),
		}
		mockService.On(
			"GetUserByUsernameAndPassword",
			context.Background(),
			expectedUser.Username,
			"password",
		).Return(expectedUser, nil)
		mockService.On(
			"CreateAuthSession",
			context.Background(),
			expectedSession.AuthSessionUserID,
		).Return(expectedSession, nil)

		e := echo.New()
		formData := url.Values{}
		formData.Set("username", expectedUser.Username)
		formData.Set("password", "password")
		req := httptest.NewRequest(
			http.MethodPost,
			"/auth/login",
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		h := NewAuthHandler(
			mockService,
			service.NewCookieService(
				[]byte(security.GenerateRandomKey(32)),
				[]byte(security.GenerateRandomKey(24)),
			),
		)

		// act
		err := h.PostLogin(c)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, "/auth/set-password", rec.Header().Get("hx-redirect"))
	})
}

func TestAuthHandler_GetLogOut(t *testing.T) {
	t.Run("success - user is logged out and redirect to landing page", func(t *testing.T) {
		// arrange
		settings.Settings = settings.NewSettings()
		mockService := new(testutil.MockUserService)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewAuthHandler(
			mockService,
			service.NewCookieService(
				[]byte(security.GenerateRandomKey(32)),
				[]byte(security.GenerateRandomKey(24)),
			),
		)

		// act
		err := h.GetLogOut(c)

		// assert
		assert.NoError(t, err)
		cookies := rec.Result().Cookies()
		assert.Equal(t, 1, len(cookies))
		cookie := cookies[0]
		assert.True(t, cookie.Expires.Before(time.Now().UTC()))
		assert.Equal(t, "", cookie.Value)
	})
}

func TestAuthHandler_GetSetPasswordPage(t *testing.T) {
	t.Run("success - set password page html is returned", func(t *testing.T) {
		// arrange
		settings.Settings = settings.NewSettings()
		mockService := new(testutil.MockUserService)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/auth/set-password", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		user := &store.User{
			UserID:            1,
			UserRoleID:        store.Operator,
			Username:          "testuser",
			PasswordHash:      "password",
			PasswordChangedOn: util.AsPtr(time.Now().UTC().Add(-30 * time.Second)),
		}
		c.Set("user", user)
		h := NewAuthHandler(
			mockService,
			service.NewCookieService(
				[]byte(security.GenerateRandomKey(32)),
				[]byte(security.GenerateRandomKey(24)),
			),
		)

		// act
		err := h.GetSetPasswordPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, `name="username"`)
		assert.Contains(t, body, `name="password"`)
		assert.Contains(t, body, `name="password_confirm"`)
	})
}

func TestAuthHandler_PostSetPassword(t *testing.T) {
	t.Run("success - user is redirected to login page", func(t *testing.T) {
		// arrange
		settings.Settings = settings.NewSettings()
		mockService := new(testutil.MockUserService)
		var userID int64 = 1
		username := "testuser"
		password := "password"
		mockService.On("SetUserPassword", context.Background(), userID, password).
			Return(nil)

		e := echo.New()
		formData := url.Values{}
		formData.Add("username", username)
		formData.Add("password", password)
		formData.Add("password_confirm", password)
		req := httptest.NewRequest(
			http.MethodPost,
			"/auth/set-password",
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		user := &store.User{
			UserID:            userID,
			UserRoleID:        store.Operator,
			Username:          username,
			PasswordHash:      "passwordhash",
			PasswordChangedOn: util.AsPtr(time.Now().UTC().Add(-30 * time.Second)),
		}
		c.Set("user", user)
		h := NewAuthHandler(
			mockService,
			service.NewCookieService(
				[]byte(security.GenerateRandomKey(32)),
				[]byte(security.GenerateRandomKey(24)),
			),
		)

		// act
		err := h.PostSetPassword(c)

		// assert
		assert.NoError(t, err)
		cookies := rec.Result().Cookies()
		assert.Equal(t, 1, len(cookies))
		cookie := cookies[0]
		assert.True(t, cookie.Expires.Before(time.Now().UTC()))
		assert.Equal(t, "", cookie.Value)
		assert.Equal(t, "/", rec.Result().Header.Get("hx-redirect"))
	})
	t.Run("failure - username != signed in username", func(t *testing.T) {
		// arrange
		settings.Settings = settings.NewSettings()
		mockService := new(testutil.MockUserService)
		var userID int64 = 1
		password := "password"
		mockService.On("SetUserPassword", context.Background(), userID, password).
			Return(nil)

		e := echo.New()
		formData := url.Values{}
		formData.Add("username", "testuser")
		formData.Add("password", password)
		formData.Add("password_confirm", password)
		req := httptest.NewRequest(
			http.MethodPost,
			"/auth/set-password",
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		user := &store.User{
			UserID:            userID,
			UserRoleID:        store.Operator,
			Username:          "signedinuser",
			PasswordHash:      "passwordhash",
			PasswordChangedOn: util.AsPtr(time.Now().UTC().Add(-30 * time.Second)),
		}
		c.Set("user", user)
		h := NewAuthHandler(
			mockService,
			service.NewCookieService(
				[]byte(security.GenerateRandomKey(32)),
				[]byte(security.GenerateRandomKey(24)),
			),
		)

		// act
		err := h.PostSetPassword(c)

		// assert
		assert.Error(t, err)
		htmxErr, ok := err.(*HTMXError)
		assert.True(t, ok)
		assert.Equal(t, "invalid username", htmxErr.Message)
	})
	t.Run("failure - password != password confirm", func(t *testing.T) {
		// arrange
		settings.Settings = settings.NewSettings()
		mockService := new(testutil.MockUserService)
		var userID int64 = 1
		username := "testuser"
		password := "password"
		passwordConfirm := "notpassword"
		mockService.On("SetUserPassword", context.Background(), userID, password).
			Return(nil)

		e := echo.New()
		formData := url.Values{}
		formData.Add("username", username)
		formData.Add("password", password)
		formData.Add("password_confirm", passwordConfirm)
		req := httptest.NewRequest(
			http.MethodPost,
			"/auth/set-password",
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		user := &store.User{
			UserID:            userID,
			UserRoleID:        store.Operator,
			Username:          username,
			PasswordHash:      "passwordhash",
			PasswordChangedOn: util.AsPtr(time.Now().UTC().Add(-30 * time.Second)),
		}
		c.Set("user", user)
		h := NewAuthHandler(
			mockService,
			service.NewCookieService(
				[]byte(security.GenerateRandomKey(32)),
				[]byte(security.GenerateRandomKey(24)),
			),
		)

		// act
		err := h.PostSetPassword(c)

		// assert
		assert.Error(t, err)
		htmxErr, ok := err.(*HTMXError)
		assert.True(t, ok)
		assert.Equal(t, "passwords do not match", htmxErr.Message)
	})
}
