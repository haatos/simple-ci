package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/views"
	"github.com/haatos/simple-ci/internal/views/pages"
	"github.com/labstack/echo/v4"
)

const (
	deleteCredentialErrorTarget string = "#delete-credential-error"
)

func SetupCredentialRoutes(g *echo.Group, credentialService CredentialServicer) {
	h := NewCredentialHandler(credentialService)
	credentialsGroup := g.Group("/app/credentials", IsAuthenticated)
	credentialsGroup.GET("", h.GetCredentialsPage)
	credentialsGroup.POST("", h.PostCredentials, RoleMiddleware(store.Admin))
	credentialsGroup.PATCH("", h.PatchCredential, RoleMiddleware(store.Admin))
	credentialsGroup.DELETE("/:credential_id", h.DeleteCredential, RoleMiddleware(store.Admin))
	credentialsGroup.GET("/:credential_id", h.GetCredentialPage)
}

type CredentialWriter interface {
	CreateCredential(
		ctx context.Context,
		username, description, sshPrivateKey string,
	) (*store.Credential, error)
	UpdateCredential(ctx context.Context, id int64, username, description string) error
	DeleteCredential(ctx context.Context, id int64) error
}

type CredentialReader interface {
	GetCredentialByID(ctx context.Context, id int64) (*store.Credential, error)
	ListCredentials(ctx context.Context) ([]*store.Credential, error)
	DecryptAES(s string) ([]byte, error)
}

type CredentialServicer interface {
	CredentialWriter
	CredentialReader
}

type CredentialHandler struct {
	credentialService CredentialServicer
}

func NewCredentialHandler(credentialService CredentialServicer) *CredentialHandler {
	return &CredentialHandler{credentialService}
}

func (h *CredentialHandler) GetCredentialsPage(c echo.Context) error {
	u := getCtxUser(c)
	credentials, err := h.credentialService.ListCredentials(c.Request().Context())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return newError(c, err,
			http.StatusInternalServerError,
			"something went wrong while listing credentials",
		)
	}
	if isHXRequest(c) {
		return render(c, pages.CredentialsMain(credentials))
	}
	return render(c, pages.CredentialsPage(u, credentials))
}

func (h *CredentialHandler) GetCredentialPage(c echo.Context) error {
	u := getCtxUser(c)
	getCredential := new(CredentialParams)
	if err := c.Bind(getCredential); err != nil {
		return newError(c, err,
			http.StatusBadRequest, "invalid credential data",
		)
	}

	credential, err := h.credentialService.GetCredentialByID(
		c.Request().Context(), getCredential.CredentialID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err,
				http.StatusNotFound, "credential was not found",
			)
		}
		return newError(c, err,
			http.StatusInternalServerError,
			"something went wrong while getting credential data",
		)
	}

	if isHXRequest(c) {
		return render(c, pages.CredentialMain(credential))
	}
	return render(c, pages.CredentialPage(u, credential))
}

func (h *CredentialHandler) PostCredentials(c echo.Context) error {
	cp := new(CredentialParams)
	if err := c.Bind(cp); err != nil {
		return newError(c, err,
			http.StatusBadRequest,
			"Invalid credential data",
		)
	}

	credential, err := h.credentialService.CreateCredential(
		c.Request().Context(), cp.Username, cp.Description, cp.SSHPrivateKey,
	)
	if err != nil {
		return newError(c, err,
			http.StatusInternalServerError,
			"Something went wrong when creating new credentials.",
		)
	}

	return render(c, pages.CredentialCard(credential))
}

func (h *CredentialHandler) PatchCredential(c echo.Context) error {
	cp := new(CredentialParams)
	if err := c.Bind(cp); err != nil {
		return newError(c, err,
			http.StatusBadRequest, "invalid credential data",
		)
	}

	err := h.credentialService.UpdateCredential(
		c.Request().Context(),
		cp.CredentialID,
		strings.TrimSpace(cp.Username),
		strings.TrimSpace(cp.Description),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err,
				http.StatusNotFound, "credential was not found",
			)
		}
		return newError(c, err,
			http.StatusInternalServerError,
			"something went wrong while updating credential",
		)
	}

	return renderToast(c, views.SuccessToast("Credential updated.", 3000))
}

func (h *CredentialHandler) DeleteCredential(c echo.Context) error {
	cp := new(CredentialParams)
	if err := c.Bind(cp); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid credential data")
	}

	if cp.CredentialID == 0 {
		return newError(c, errors.New("credential id was zero"),
			http.StatusBadRequest, "invalid credential ID",
		)
	}

	credential, err := h.credentialService.GetCredentialByID(
		c.Request().Context(), cp.CredentialID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusNotFound, "credential not found")
		}
		return newError(c, err, http.StatusNotFound, "unable to delete credential")
	}

	if err := h.credentialService.DeleteCredential(
		c.Request().Context(), credential.CredentialID,
	); err != nil {
		if isForeignKeyConstraintError(err) {
			return newErrorString(
				c, err, http.StatusBadRequest,
				deleteCredentialErrorTarget,
				"credential is in use",
			)
		}
		return newErrorString(
			c, err, http.StatusBadRequest,
			deleteCredentialErrorTarget,
			"unable to delete credential",
		)
	}

	return hxRedirect(c, "/app/credentials")
}
