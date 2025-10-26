package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/haatos/simple-ci/internal/service"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/views"
	"github.com/haatos/simple-ci/internal/views/pages"
	"github.com/labstack/echo/v4"
)

func SetupAPIKeyRoutes(g *echo.Group, apiKeyService service.APIKeyServicer) {
	h := NewAPIKeyHandler(apiKeyService)
	apiKeysGroup := g.Group("/app/api-keys", IsAuthenticated)
	apiKeysGroup.GET("", h.GetAPIKeysPage, RoleMiddleware(store.Admin))
	apiKeysGroup.POST("", h.PostAPIKey, RoleMiddleware(store.Admin))
	apiKeysGroup.DELETE("/:id", h.DeleteAPIKey, RoleMiddleware(store.Admin))
}

type APIKeyHandler struct {
	apiKeyService service.APIKeyServicer
}

func NewAPIKeyHandler(apiKeyService service.APIKeyServicer) *APIKeyHandler {
	return &APIKeyHandler{apiKeyService}
}

func (h *APIKeyHandler) GetAPIKeysPage(c echo.Context) error {
	u := getCtxUser(c)
	apiKeys, err := h.apiKeyService.ListAPIKeys(c.Request().Context())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return newError(c, err,
			http.StatusInternalServerError,
			"something went wrong while listing credentials",
		)
	}
	if isHXRequest(c) {
		return render(c, pages.APIKeysMain(apiKeys))
	}
	return render(c, pages.APIKeysPage(u, apiKeys))
}

func (h *APIKeyHandler) PostAPIKey(c echo.Context) error {
	ak, err := h.apiKeyService.CreateAPIKey(c.Request().Context())
	if err != nil {
		return newError(
			c, err,
			http.StatusInternalServerError, "unable to create api key",
		)
	}

	return render(c, pages.APIKeyRow(ak))
}

func (h *APIKeyHandler) DeleteAPIKey(c echo.Context) error {
	akp := new(APIKeyParams)
	if err := c.Bind(akp); err != nil {
		return newError(
			c, err,
			http.StatusBadRequest, "invalid api key data",
		)
	}

	if err := h.apiKeyService.DeleteAPIKey(c.Request().Context(), akp.ID); err != nil {
		return newError(
			c, err,
			http.StatusInternalServerError, "unable to delete api key",
		)
	}

	return renderToast(c, views.SuccessToast("API key deleted", 3000))
}
