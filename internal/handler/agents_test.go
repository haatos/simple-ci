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

	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAgentsHandler_GetAgentsPage(t *testing.T) {
	t.Run("success - agents found on page", func(t *testing.T) {
		// arrange
		credential := generateCredential()
		agent := generateAgent(credential.CredentialID)
		expectedAgents := []*store.Agent{agent}
		expectedCredentials := []*store.Credential{credential}
		mockAgentService := new(testutil.MockAgentService)
		mockAgentService.On("ListAgentsAndCredentials", context.Background()).
			Return(expectedAgents, expectedCredentials, nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app/agents", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewAgentHandler(mockAgentService)

		// act
		err := h.GetAgentsPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `<h2 class="card-title"`)
		assert.Contains(t, body, agent.Name)
		assert.Contains(t, body, `<p class="text-lg"`)
		assert.Contains(t, body, agent.Description)
	})
	t.Run("success - agents found on main element", func(t *testing.T) {
		// arrange
		credential := generateCredential()
		agent := generateAgent(credential.CredentialID)
		expectedAgents := []*store.Agent{agent}
		expectedCredentials := []*store.Credential{credential}
		mockAgentService := new(testutil.MockAgentService)
		mockAgentService.On("ListAgentsAndCredentials", context.Background()).
			Return(expectedAgents, expectedCredentials, nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app/agents", nil)
		req.Header.Set("hx-request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewAgentHandler(mockAgentService)

		// act
		err := h.GetAgentsPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.NotContains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `<h2 class="card-title"`)
		assert.Contains(t, body, agent.Name)
		assert.Contains(t, body, `<p class="text-lg"`)
		assert.Contains(t, body, agent.Description)
	})
}

func TestAgentsHandler_GetAgentPage(t *testing.T) {
	t.Run("success - agent form found on page", func(t *testing.T) {
		// arrange
		credential := generateCredential()
		expectedAgent := generateAgent(credential.CredentialID)
		expectedCredentials := []*store.Credential{credential}
		mockAgentService := new(testutil.MockAgentService)
		mockAgentService.On(
			"GetAgentAndCredentials",
			context.Background(),
			expectedAgent.AgentID,
		).Return(expectedAgent, expectedCredentials, nil)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/app/agents/%d", expectedAgent.AgentID),
			nil,
		)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("agent_id")
		c.SetParamValues(fmt.Sprintf("%d", expectedAgent.AgentID))
		h := NewAgentHandler(mockAgentService)

		// act
		err := h.GetAgentPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `name="name"`)
		assert.Contains(t, body, `name="hostname"`)
		assert.Contains(t, body, `name="workspace"`)
		assert.Contains(t, body, `name="description"`)
	})
	t.Run("success - agent form found on main element", func(t *testing.T) {
		// arrange
		credential := generateCredential()
		expectedAgent := generateAgent(credential.CredentialID)
		expectedCredentials := []*store.Credential{credential}
		mockAgentService := new(testutil.MockAgentService)
		mockAgentService.On(
			"GetAgentAndCredentials",
			context.Background(),
			expectedAgent.AgentID,
		).Return(expectedAgent, expectedCredentials, nil)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodGet,
			fmt.Sprintf("/app/agents/%d", expectedAgent.AgentID),
			nil,
		)
		req.Header.Set("hx-request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("agent_id")
		c.SetParamValues(fmt.Sprintf("%d", expectedAgent.AgentID))
		h := NewAgentHandler(mockAgentService)

		// act
		err := h.GetAgentPage(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.NotContains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, `name="name"`)
		assert.Contains(t, body, `name="hostname"`)
		assert.Contains(t, body, `name="workspace"`)
		assert.Contains(t, body, `name="description"`)
	})
}

func TestAgentsHandler_PatchAgent(t *testing.T) {
	t.Run("success - agent updated", func(t *testing.T) {
		// arrange
		agent := generateAgent(0)
		mockAgentService := new(testutil.MockAgentService)
		mockAgentService.On(
			"UpdateAgent",
			context.Background(),
			agent.AgentID, agent.AgentCredentialID,
			agent.Name, agent.Hostname, agent.Workspace, agent.Description, agent.OSType,
		).Return(nil)

		formData := url.Values{}
		formData.Set("agent_credential_id", fmt.Sprintf("%d", agent.AgentCredentialID))
		formData.Set("name", agent.Name)
		formData.Set("hostname", agent.Hostname)
		formData.Set("workspace", agent.Workspace)
		formData.Set("description", agent.Description)
		formData.Set("os_type", agent.OSType)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodPatch,
			fmt.Sprintf("/app/agents/%d", agent.AgentID),
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("agent_id")
		c.SetParamValues(fmt.Sprintf("%d", agent.AgentID))
		h := NewAgentHandler(mockAgentService)

		// act
		err := h.PatchAgent(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, `name="success-toast"`)
		assert.Contains(t, body, "Agent updated")
	})
	t.Run("failure - agent not found", func(t *testing.T) {
		// arrange
		agent := generateAgent(0)
		mockAgentService := new(testutil.MockAgentService)
		mockAgentService.On(
			"UpdateAgent",
			context.Background(),
			agent.AgentID,
			agent.AgentCredentialID,
			agent.Name,
			agent.Hostname,
			agent.Workspace,
			agent.Description,
			agent.OSType,
		).Return(sql.ErrNoRows)

		formData := url.Values{}
		formData.Set("agent_credential_id", fmt.Sprintf("%d", agent.AgentCredentialID))
		formData.Set("name", agent.Name)
		formData.Set("hostname", agent.Hostname)
		formData.Set("workspace", agent.Workspace)
		formData.Set("description", agent.Description)
		formData.Set("os_type", agent.OSType)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodPatch,
			fmt.Sprintf("/app/agents/%d", agent.AgentID),
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		c.SetParamNames("agent_id")
		c.SetParamValues(fmt.Sprintf("%d", agent.AgentID))
		h := NewAgentHandler(mockAgentService)

		// act
		err := h.PatchAgent(c)

		// assert
		assert.Error(t, err)
		htmxErr, ok := err.(*HTMXError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusNotFound, htmxErr.Code)
	})
}

func TestAgentsHandler_DeleteAgent(t *testing.T) {
	t.Run("success - agent deleted", func(t *testing.T) {
		// arrange
		credential := generateCredential()
		expectedAgent := generateAgent(credential.CredentialID)
		mockAgentService := new(testutil.MockAgentService)
		mockAgentService.On(
			"GetAgentByID",
			context.Background(),
			expectedAgent.AgentID,
		).Return(expectedAgent, nil)
		mockAgentService.On(
			"DeleteAgent",
			context.Background(),
			expectedAgent.AgentID,
		).Return(nil)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("/app/agents/%d", expectedAgent.AgentID),
			nil,
		)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("agent_id")
		c.SetParamValues(fmt.Sprintf("%d", expectedAgent.AgentID))
		h := NewAgentHandler(mockAgentService)

		// act
		err := h.DeleteAgent(c)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, "/app/agents", c.Response().Header().Get("hx-redirect"))
	})
	t.Run("failure - agent not found", func(t *testing.T) {
		// arrange
		agent := generateAgent(0)
		mockAgentService := new(testutil.MockAgentService)
		mockAgentService.On("GetAgentByID", context.Background(), agent.AgentID).
			Return(nil, sql.ErrNoRows)
		mockAgentService.On("DeleteAgent", context.Background(), agent.AgentID).
			Return(sql.ErrNoRows)

		e := echo.New()
		req := httptest.NewRequest(
			http.MethodDelete,
			fmt.Sprintf("/app/agents/%d", agent.AgentID),
			nil,
		)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		c.SetParamNames("agent_id")
		c.SetParamValues(fmt.Sprintf("%d", agent.AgentID))
		h := NewAgentHandler(mockAgentService)

		// act
		err := h.DeleteAgent(c)

		// assert
		assert.Error(t, err)
		htmxErr, ok := err.(*HTMXError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusNotFound, htmxErr.Code)
	})
}

func generateAgent(credentialID int64) *store.Agent {
	if credentialID == 0 {
		credentialID = rand.Int63()
	}
	a := &store.Agent{
		AgentID:           rand.Int63(),
		AgentCredentialID: &credentialID,
		Name:              fmt.Sprintf("agent%d", time.Now().UnixNano()),
		Hostname:          "localhost",
		Workspace:         "/tmp",
		Description:       fmt.Sprintf("description%d", time.Now().UnixNano()),
	}
	return a
}
