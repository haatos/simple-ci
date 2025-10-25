package handler

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal/security"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestCredentialsHandler_GetCredentialsPage(t *testing.T) {
	t.Run("success - credentials found on page", func(t *testing.T) {
		// arrange
		mockService := new(testutil.MockCredentialService)
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app/credentials", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewCredentialHandler(mockService)
		credential := generateCredential()
		expectedCredentials := []*store.Credential{credential}
		mockService.On(
			"ListCredentials", c.Request().Context(),
		).Return(expectedCredentials, nil)

		// act
		err := h.GetCredentialsPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `<h2 class="card-title"`)
		assert.Contains(t, body, credential.Username)
		assert.Contains(t, body, `<p class="text-lg"`)
		assert.Contains(t, body, credential.Description)
	})
	t.Run("success - credentials found on main element", func(t *testing.T) {
		// arrange
		credential := generateCredential()
		expectedCredentials := []*store.Credential{credential}
		mockService := new(testutil.MockCredentialService)
		mockService.On("ListCredentials", context.Background()).Return(expectedCredentials, nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app/credentials", nil)
		req.Header.Set("hx-request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewCredentialHandler(mockService)

		// act
		err := h.GetCredentialsPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.NotContains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `<h2 class="card-title"`)
		assert.Contains(t, body, credential.Username)
		assert.Contains(t, body, `<p class="text-lg"`)
		assert.Contains(t, body, credential.Description)
	})
}

func TestCredentialsHandler_GetCredentialPage(t *testing.T) {
	t.Run("success - credential form found on page", func(t *testing.T) {
		// arrange
		expectedCredential := generateCredential()
		mockService := new(testutil.MockCredentialService)
		mockService.On(
			"GetCredentialByID",
			context.Background(),
			expectedCredential.CredentialID,
		).Return(expectedCredential, nil)
		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/app/credentials/%d", expectedCredential.CredentialID),
			nil,
		)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("credential_id")
		c.SetParamValues(fmt.Sprintf("%d", expectedCredential.CredentialID))
		h := NewCredentialHandler(mockService)

		// act
		err := h.GetCredentialPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `name="username"`)
		assert.Contains(t, body, `name="description"`)
	})
	t.Run("success - credentials form found on main element", func(t *testing.T) {
		// arrange
		expectedCredential := generateCredential()
		mockService := new(testutil.MockCredentialService)
		mockService.On(
			"GetCredentialByID",
			context.Background(),
			expectedCredential.CredentialID,
		).Return(expectedCredential, nil)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/app/credentials/%d", expectedCredential.CredentialID),
			nil,
		)
		req.Header.Set("hx-request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("credential_id")
		c.SetParamValues(fmt.Sprintf("%d", expectedCredential.CredentialID))
		h := NewCredentialHandler(mockService)

		// act
		err := h.GetCredentialPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.NotContains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `name="username"`)
		assert.Contains(t, body, `name="description"`)
	})
}

func TestCredentialsHandler_PatchCredential(t *testing.T) {
	t.Run("success - credential updated", func(t *testing.T) {
		// arrange
		credential := generateCredential()
		mockService := new(testutil.MockCredentialService)
		mockService.On(
			"UpdateCredential",
			context.Background(),
			credential.CredentialID,
			credential.Username,
			credential.Description,
		).Return(nil)

		formData := url.Values{}
		formData.Set("username", credential.Username)
		formData.Set("description", credential.Description)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodPatch,
			fmt.Sprintf("/app/credentials/%d", credential.CredentialID),
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		c.SetParamNames("credential_id")
		c.SetParamValues(fmt.Sprintf("%d", credential.CredentialID))
		h := NewCredentialHandler(mockService)

		// act
		err := h.PatchCredential(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, `name="success-toast"`)
		assert.Contains(t, body, "Credential updated")
	})
	t.Run("failure - credential not found", func(t *testing.T) {
		// arrange
		credential := generateCredential()
		mockService := new(testutil.MockCredentialService)
		mockService.On(
			"UpdateCredential",
			context.Background(),
			credential.CredentialID,
			credential.Username,
			credential.Description,
		).Return(sql.ErrNoRows)

		formData := url.Values{}
		formData.Set("username", credential.Username)
		formData.Set("description", credential.Description)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodPatch,
			fmt.Sprintf("/app/credentials/%d", credential.CredentialID),
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		c.SetParamNames("credential_id")
		c.SetParamValues(fmt.Sprintf("%d", credential.CredentialID))
		h := NewCredentialHandler(mockService)

		// act
		err := h.PatchCredential(c)

		// assert
		assert.Error(t, err)
		htmxErr, ok := err.(*HTMXError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusNotFound, htmxErr.Code)
	})
}

func TestCredentialsHandler_DeleteCredential(t *testing.T) {
	t.Run("success - credential deleted", func(t *testing.T) {
		// arrange
		expectedCredential := generateCredential()
		mockService := new(testutil.MockCredentialService)
		mockService.On(
			"GetCredentialByID",
			context.Background(),
			expectedCredential.CredentialID,
		).Return(expectedCredential, nil)
		mockService.On(
			"DeleteCredential",
			context.Background(),
			expectedCredential.CredentialID,
		).Return(nil)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("/app/credentials/%d", expectedCredential.CredentialID),
			nil,
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		c.SetParamNames("credential_id")
		c.SetParamValues(fmt.Sprintf("%d", expectedCredential.CredentialID))
		h := NewCredentialHandler(mockService)

		// act
		err := h.DeleteCredential(c)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, "/app/credentials", c.Response().Header().Get("hx-redirect"))
	})
	t.Run("failure - credential not found", func(t *testing.T) {
		// arrange
		credential := generateCredential()
		mockService := new(testutil.MockCredentialService)
		mockService.On(
			"GetCredentialByID",
			context.Background(),
			credential.CredentialID,
		).Return(nil, sql.ErrNoRows)
		mockService.On(
			"DeleteCredential",
			context.Background(),
			credential.CredentialID,
		).Return(sql.ErrNoRows)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("/app/credentials/%d", credential.CredentialID),
			nil,
		)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		c.SetParamNames("credential_id")
		c.SetParamValues(fmt.Sprintf("%d", credential.CredentialID))
		h := NewCredentialHandler(mockService)

		// act
		err := h.DeleteCredential(c)

		// assert
		assert.Error(t, err)
		htmxErr, ok := err.(*HTMXError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusNotFound, htmxErr.Code)
	})
}

func generateCredential() *store.Credential {
	return &store.Credential{
		CredentialID:      rand.Int63(),
		Username:          fmt.Sprintf("testuser%d", time.Now().UnixNano()),
		Description:       fmt.Sprintf("description%d", time.Now().UnixNano()),
		SSHPrivateKeyHash: security.GenerateRandomKey(32),
	}
}
