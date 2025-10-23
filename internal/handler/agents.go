package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/haatos/simple-ci/internal/service"
	"github.com/haatos/simple-ci/internal/views"
	"github.com/haatos/simple-ci/internal/views/pages"
	"github.com/labstack/echo/v4"
)

const (
	deleteAgentErrorTarget string = "#delete-agent-error"
)

type AgentHandler struct {
	agentService service.AgentServicer
}

func NewAgentHandler(
	agentService service.AgentServicer,
) *AgentHandler {
	return &AgentHandler{agentService}
}

func (h *AgentHandler) GetAgentsPage(c echo.Context) error {
	u := getCtxUser(c)
	agents, credentials, err := h.agentService.ListAgentsAndCredentials(c.Request().Context())
	if err != nil {
		return newError(c, err,
			http.StatusInternalServerError,
			"Something went wrong listing agents/credentials",
		)
	}

	if isHXRequest(c) {
		return render(c, pages.AgentsMain(agents, credentials))
	}
	return render(c, pages.AgentsPage(u, agents, credentials))
}

func (h *AgentHandler) PostAgent(c echo.Context) error {
	ap := new(AgentParams)
	if err := c.Bind(ap); err != nil {
		return newError(c, err,
			http.StatusBadRequest, "invalid agent data",
		)
	}

	ap.Name = strings.TrimSpace(ap.Name)
	ap.Hostname = strings.TrimSpace(ap.Hostname)
	ap.Workspace = strings.TrimSpace(ap.Workspace)
	ap.Description = strings.TrimSpace(ap.Description)

	a, err := h.agentService.CreateAgent(
		c.Request().Context(),
		ap.AgentCredentialID,
		ap.Name,
		ap.Hostname,
		ap.Workspace,
		ap.Description,
		ap.OSType,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return newError(
				c,
				err,
				http.StatusConflict,
				fmt.Sprintf("An agent with the name %s already exists", a.Name),
			)
		} else {
			return newError(c, err, http.StatusInternalServerError, "unable to create agent")
		}
	}

	return render(c, pages.AgentCard(a))
}

func (h *AgentHandler) GetAgentPage(c echo.Context) error {
	u := getCtxUser(c)
	ap := new(AgentParams)
	if err := c.Bind(ap); err != nil {
		return newError(c, err,
			http.StatusBadRequest, "invalid agent data",
		)
	}

	agent, credentials, err := h.agentService.GetAgentAndCredentials(
		c.Request().Context(),
		ap.AgentID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err,
				http.StatusNotFound, "agent was not found",
			)
		}
		return newError(c, err,
			http.StatusInternalServerError,
			"something went wrong while getting agent data",
		)
	}

	if isHXRequest(c) {
		return render(c, pages.AgentMain(agent, credentials))
	}
	return render(c, pages.AgentPage(u, agent, credentials))
}

func (h *AgentHandler) PatchAgent(c echo.Context) error {
	ap := new(AgentParams)
	if err := c.Bind(ap); err != nil {
		return newError(c, err,
			http.StatusBadRequest, "invalid agent data",
		)
	}

	if err := h.agentService.UpdateAgent(
		c.Request().Context(),
		ap.AgentID,
		ap.AgentCredentialID,
		ap.Name,
		ap.Hostname,
		ap.Workspace,
		ap.Description,
		ap.OSType,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err,
				http.StatusNotFound, "agent was not found",
			)
		}
		return newError(c, err,
			http.StatusInternalServerError,
			"something went wrong while updating agent",
		)
	}

	return renderToast(c, views.SuccessToast("Agent updated", 3000))
}

func (h *AgentHandler) DeleteAgent(c echo.Context) error {
	ap := new(AgentParams)
	if err := c.Bind(ap); err != nil || ap.AgentID == 0 {
		return newError(c, err, http.StatusBadRequest, "invalid agent ID")
	}

	a, err := h.agentService.GetAgentByID(c.Request().Context(), ap.AgentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusNotFound, "agent not found")
		}
		return newError(c, err, http.StatusInternalServerError, "unable to delete agent")
	}

	if err := h.agentService.DeleteAgent(c.Request().Context(), a.AgentID); err != nil {
		message := "unable to delete agent"
		if isForeignKeyConstraintError(err) {
			message = "agent is in use"
		}
		return newErrorString(
			c, err, http.StatusBadRequest,
			deleteAgentErrorTarget, message,
		)
	}

	return hxRedirect(c, "/app/agents")
}

func (h *AgentHandler) PostTestAgentConnection(c echo.Context) error {
	ap := new(AgentParams)
	if err := c.Bind(ap); err != nil || ap.AgentID == 0 {
		return newError(c, err, http.StatusBadRequest, "invalid agent ID")
	}

	if err := h.agentService.TestAgentConnection(
		c.Request().Context(), ap.AgentID,
	); err != nil {
		return newError(c, err,
			http.StatusInternalServerError,
			"testing agent connection failed, check logs for details",
		)
	}

	return renderToast(c, views.SuccessToast("connection ok", 3000))
}
