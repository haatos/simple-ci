package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAppHandler_GetAppPage(t *testing.T) {
	testcases := []struct {
		role         store.Role
		contained    []string
		notContained []string
	}{
		{
			role: store.Operator,
			contained: []string{
				"<html",
				"<main",
				`href="/app/credentials"`,
				`href="/app/agents"`,
				`href="/app/pipelines"`,
			},
			notContained: []string{
				`href="/app/users"`,
				`href="/app/api-keys"`,
			},
		},
		{
			role: store.Admin,
			contained: []string{
				"<html",
				"<main",
				`href="/app/credentials"`,
				`href="/app/agents"`,
				`href="/app/pipelines"`,
				`href="/app/api-keys"`,
			},
			notContained: []string{
				`href="/app/users"`,
			},
		},
		{
			role: store.Superuser,
			contained: []string{
				"<html",
				"<main",
				`href="/app/credentials"`,
				`href="/app/agents"`,
				`href="/app/pipelines"`,
				`href="/app/api-keys"`,
				`href="/app/users"`,
			},
			notContained: []string{},
		},
	}
	for _, tc := range testcases {
		t.Run(fmt.Sprintf("app page html for %s", tc.role.ToString()), func(t *testing.T) {
			// arrange
			user := generateUser(tc.role, nil, nil)
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/app", nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("user", user)

			// act
			err := GetAppPage(c)

			// assert
			assert.NoError(t, err)
			body := rec.Body.String()
			for _, s := range tc.contained {
				assert.Contains(t, body, s)
			}
			for _, s := range tc.notContained {
				assert.NotContains(t, body, s)
			}
		})
	}
}
