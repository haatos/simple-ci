package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/views"
	"github.com/haatos/simple-ci/internal/views/pages"
	"github.com/labstack/echo/v4"
)

func SetupAPIKeyRoutes(g *echo.Group, apiKeyService APIKeyServicer) {
	h := NewAPIKeyHandler(apiKeyService)
	apiKeysGroup := g.Group("/app/api-keys", IsAuthenticated)
	apiKeysGroup.GET("", h.GetAPIKeysPage, RoleMiddleware(store.Admin))
	apiKeysGroup.POST("", h.PostAPIKey, RoleMiddleware(store.Admin))
	apiKeysGroup.DELETE("/:id", h.DeleteAPIKey, RoleMiddleware(store.Admin))
}

type APIKeyWriter interface {
	CreateAPIKey(ctx context.Context) (*store.APIKey, error)
	DeleteAPIKey(ctx context.Context, id int64) error
}

type APIKeyReader interface {
	GetAPIKeyByID(ctx context.Context, id int64) (*store.APIKey, error)
	GetAPIKeyByValue(ctx context.Context, value string) (*store.APIKey, error)
	ListAPIKeys(ctx context.Context) ([]*store.APIKey, error)
}

type APIKeyServicer interface {
	APIKeyWriter
	APIKeyReader
}

type APIKeyHandler struct {
	apiKeyService APIKeyServicer
}

func NewAPIKeyHandler(apiKeyService APIKeyServicer) *APIKeyHandler {
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
