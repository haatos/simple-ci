package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/haatos/simple-ci/internal"
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

func TestAppHandler_GetConfigPage(t *testing.T) {
	t.Run("success - config page html is returned", func(t *testing.T) {
		// arrange
		internal.InitializeConfiguration()
		defer os.Remove("config.json")
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app/config", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", &store.User{UserID: 1, UserRoleID: types.Operator, Username: "testuser"})

		// act
		err := GetConfigPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<html")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `name="session_expires_hours"`)
		assert.Contains(t, body, `name="queue_size"`)
	})
	t.Run("success - config page main html is returned", func(t *testing.T) {
		// arrange
		internal.InitializeConfiguration()
		defer os.Remove("config.json")
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app/config", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		req.Header.Set("hx-request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", &store.User{UserID: 1, UserRoleID: types.Operator, Username: "testuser"})

		// act
		err := GetConfigPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.NotContains(t, body, "<html")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `name="session_expires_hours"`)
		assert.Contains(t, body, `name="queue_size"`)
	})
}

func TestAppHandler_PostConfiguration(t *testing.T) {
	t.Run("success - configuration is updated", func(t *testing.T) {
		// arrange
		internal.Config = &internal.Configuration{
			SessionExpiresHours: internal.NewHoursDuration(24),
			QueueSize:           4,
		}
		defer os.Remove("config.json")
		var sessionExpiresHours int64 = 360
		var queueSize int64 = 6

		formData := url.Values{}
		formData.Add("session_expires_hours", fmt.Sprintf("%d", sessionExpiresHours))
		formData.Add("queue_size", fmt.Sprintf("%d", queueSize))
		req := httptest.NewRequest(
			http.MethodPost,
			"/app/config",
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set("hx-request", "true")
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		e := echo.New()
		c := e.NewContext(req, rec)

		// act
		err := PostConfig(c)

		// assert
		assert.NoError(t, err)
		assert.Equal(
			t,
			internal.NewHoursDuration(sessionExpiresHours),
			internal.Config.SessionExpiresHours,
		)
		assert.Equal(t, queueSize, internal.Config.QueueSize)
	})
}
