package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/types"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAppHandler_GetAppPage(t *testing.T) {
	t.Run("success - app page html is returned for operator user", func(t *testing.T) {
		// arrange
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", &store.User{UserID: 1, UserRoleID: types.Operator, Username: "testuser"})

		// act
		err := GetAppPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<html")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `href="/app/credentials"`)
		assert.Contains(t, body, `href="/app/agents"`)
		assert.Contains(t, body, `href="/app/pipelines"`)
		assert.NotContains(t, body, `href="/app/users"`)
	})
	t.Run("success - app page main html is returned for operator user", func(t *testing.T) {
		// arrange
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", &store.User{UserID: 1, UserRoleID: types.Operator, Username: "testuser"})
		c.Request().Header.Set("hx-request", "true")

		// act
		err := GetAppPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.NotContains(t, body, "<html")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `href="/app/credentials"`)
		assert.Contains(t, body, `href="/app/agents"`)
		assert.Contains(t, body, `href="/app/pipelines"`)
	})
	t.Run("success - app page html is returned for superuser", func(t *testing.T) {
		// arrange
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", &store.User{UserID: 1, UserRoleID: types.Superuser, Username: "testuser"})

		// act
		err := GetAppPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<html")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `href="/app/credentials"`)
		assert.Contains(t, body, `href="/app/agents"`)
		assert.Contains(t, body, `href="/app/pipelines"`)
		assert.Contains(t, body, `href="/app/users"`)
	})
}
