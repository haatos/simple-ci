package handler

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAPIKeysHandler_GetAPIKeysPage(t *testing.T) {
	t.Run("success - api keys found on page", func(t *testing.T) {
		// arrange
		ak := generateAPIKey()
		ctx := context.Background()
		expectedAPIKeys := []*store.APIKey{ak}
		mockService := new(testutil.MockAPIKeyService)
		mockService.On(
			"ListAPIKeys", ctx,
		).Return(expectedAPIKeys, nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app/api-keys", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewAPIKeyHandler(mockService)

		// act
		err := h.GetAPIKeysPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, fmt.Sprintf(`<td>%d</td>`, ak.ID))
		assert.Contains(t, body, fmt.Sprintf(`<td>%s`, ak.Value))
	})
	t.Run("success - api keys found on main element", func(t *testing.T) {
		// arrange
		ak := generateAPIKey()
		ctx := context.Background()
		expectedAPIKeys := []*store.APIKey{ak}
		mockService := new(testutil.MockAPIKeyService)
		mockService.On(
			"ListAPIKeys", ctx,
		).Return(expectedAPIKeys, nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app/api-keys", nil)
		req.Header.Set("hx-request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewAPIKeyHandler(mockService)

		// act
		err := h.GetAPIKeysPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.NotContains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, fmt.Sprintf(`<td>%d</td>`, ak.ID))
		assert.Contains(t, body, fmt.Sprintf(`<td>%s`, ak.Value))
	})
}

func TestAPIKeysHandler_PostAPIKey(t *testing.T) {
	t.Run("success - api key is created", func(t *testing.T) {
		// arrange
		ak := generateAPIKey()
		ctx := context.Background()
		mockService := new(testutil.MockAPIKeyService)
		mockService.On(
			"CreateAPIKey", ctx,
		).Return(ak, nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/app/api-keys", nil)
		req.Header.Set("hx-request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewAPIKeyHandler(mockService)

		// act
		err := h.PostAPIKey(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, fmt.Sprintf(`<td>%d</td>`, ak.ID))
		assert.Contains(t, body, fmt.Sprintf(`<td>%s`, ak.Value))
	})
}

func TestAPIKeysHandler_DeleteAPIKey(t *testing.T) {
	t.Run("success - api key is deleted", func(t *testing.T) {
		// arrange
		ak := generateAPIKey()
		ctx := context.Background()
		mockService := new(testutil.MockAPIKeyService)
		mockService.On(
			"DeleteAPIKey", ctx, ak.ID,
		).Return(nil)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodDelete, fmt.Sprintf("/app/api-keys/%d", ak.ID), nil,
		)
		req.Header.Set("hx-request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprintf("%d", ak.ID))
		h := NewAPIKeyHandler(mockService)

		// act
		err := h.DeleteAPIKey(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "success-toast")
		assert.Contains(t, body, "alert-success")
		assert.Contains(t, body, "API key deleted")
	})
}

func generateAPIKey() *store.APIKey {
	return &store.APIKey{
		ID:        rand.Int63(),
		Value:     uuid.NewString(),
		CreatedOn: time.Now().UTC(),
	}
}
