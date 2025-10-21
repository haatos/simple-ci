package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal/types"
	"github.com/haatos/simple-ci/internal/util"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware_AlreadyLoggedIn(t *testing.T) {
	t.Run("user is redirected to app page", func(t *testing.T) {
		// arrange
		user := generateUser(types.Admin, nil, util.AsPtr(time.Now().UTC().Add(30*time.Second)))
		user.PasswordChangedOn = util.AsPtr(time.Now().UTC().Add(-30 * time.Second))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		h := AlreadyLoggedIn(func(c echo.Context) error {
			return c.String(http.StatusOK, "not logged in")
		})
		e := echo.New()
		c := e.NewContext(req, rec)
		c.Set("user", user)

		// act
		err := h(c)

		// assert
		assert.NoError(t, err)
		url := rec.Header().Get(echo.HeaderLocation)
		assert.Equal(t, "/app", url)
	})
	t.Run("user is not logged in", func(t *testing.T) {
		// arrange
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		h := AlreadyLoggedIn(func(c echo.Context) error {
			return c.String(http.StatusOK, "not logged in")
		})
		e := echo.New()
		c := e.NewContext(req, rec)

		// act
		err := h(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Equal(t, "not logged in", body)
	})
	t.Run("user has not changed password", func(t *testing.T) {
		// arrange
		user := generateUser(types.Admin, nil, util.AsPtr(time.Now().UTC().Add(30*time.Second)))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		h := AlreadyLoggedIn(func(c echo.Context) error {
			return c.String(http.StatusOK, "not logged in")
		})
		e := echo.New()
		c := e.NewContext(req, rec)
		c.Set("user", user)

		// act
		err := h(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Equal(t, "not logged in", body)
	})
}

func TestMiddleware_IsAuthenticated(t *testing.T) {
	t.Run("user is authenticated", func(t *testing.T) {
		// arrange
		user := generateUser(types.Admin, nil, util.AsPtr(time.Now().UTC().Add(30*time.Second)))
		user.PasswordChangedOn = util.AsPtr(time.Now().UTC().Add(-30 * time.Second))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		h := IsAuthenticated(func(c echo.Context) error {
			return c.String(http.StatusOK, "is authenticated")
		})
		e := echo.New()
		c := e.NewContext(req, rec)
		c.Set("user", user)

		// act
		err := h(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Equal(t, "is authenticated", body)
	})
	t.Run("user is not authenticated", func(t *testing.T) {
		// arrange
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		h := IsAuthenticated(func(c echo.Context) error {
			return c.String(http.StatusOK, "is authenticated")
		})
		e := echo.New()
		c := e.NewContext(req, rec)

		// act
		err := h(c)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, c.Response().Status)
		assert.Equal(t, "/", c.Response().Header().Get(echo.HeaderLocation))
	})
	t.Run("user has not changed password", func(t *testing.T) {
		// arrange
		user := generateUser(types.Admin, nil, util.AsPtr(time.Now().UTC().Add(30*time.Second)))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		h := IsAuthenticated(func(c echo.Context) error {
			return c.String(http.StatusOK, "is authenticated")
		})
		e := echo.New()
		c := e.NewContext(req, rec)
		c.Set("user", user)

		// act
		err := h(c)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, c.Response().Status)
		assert.Equal(t, "/auth/set-password", c.Response().Header().Get(echo.HeaderLocation))
	})
}

func TestMiddleware_RoleMiddlware(t *testing.T) {
	t.Run("success - user has sufficient role", func(t *testing.T) {
		// arrange
		userRole := types.Admin
		requiredRole := types.Admin
		user := generateUser(userRole, nil, util.AsPtr(time.Now().UTC().Add(30*time.Second)))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		h := RoleMiddleware(requiredRole)(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})
		e := echo.New()
		c := e.NewContext(req, rec)
		c.Set("user", user)

		// act
		err := h(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Equal(t, "success", body)
	})
	t.Run("failure - user role insufficient", func(t *testing.T) {
		// arrange
		userRole := types.Operator
		requiredRole := types.Superuser
		user := generateUser(userRole, nil, util.AsPtr(time.Now().UTC().Add(30*time.Second)))
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()
		h := RoleMiddleware(requiredRole)(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})
		e := echo.New()
		c := e.NewContext(req, rec)
		c.Set("user", user)

		t.Logf("required role: %v, user role: %v\n", requiredRole, userRole)

		// act
		err := h(c)

		// assert
		assert.Error(t, err)
		echoErr := new(echo.HTTPError)
		echoErr, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusForbidden, echoErr.Code)
	})
}
