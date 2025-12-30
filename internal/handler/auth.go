package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/views/pages"

	"github.com/labstack/echo/v4"
)

type AuthCookieServicer interface {
	SetSessionCookie(echo.Context, string) error
	RemoveSessionCookie(echo.Context)
}

func SetupAuthRoutes(
	g *echo.Group,
	userService UserAuthServicer,
	cookieService AuthCookieServicer,
) {
	h := NewAuthHandler(userService, cookieService)
	g.GET("", h.GetLoginPage, AlreadyLoggedIn)
	g.GET("/auth/logout", h.GetLogOut)
	g.POST("/auth/login", h.PostLogin)
	g.GET("/auth/set-password", h.GetSetPasswordPage)
	g.POST("/auth/set-password", h.PostSetPassword)
}

type UserAuthServicer interface {
	CreateAuthSession(
		ctx context.Context,
		userID int64,
	) (*store.AuthSession, error)
	GetUserByUsernameAndPassword(
		ctx context.Context,
		username, password string,
	) (*store.User, error)
	SetUserPassword(
		ctx context.Context,
		userID int64,
		newPassword string,
	) error
}

type AuthHandler struct {
	userService   UserAuthServicer
	cookieService AuthCookieServicer
}

func NewAuthHandler(
	userService UserAuthServicer,
	cookieService AuthCookieServicer,
) *AuthHandler {
	return &AuthHandler{userService, cookieService}
}

func (h *AuthHandler) GetLoginPage(c echo.Context) error {
	u := getCtxUser(c)
	if u != nil {
		return c.Redirect(http.StatusSeeOther, "/app")
	}
	if isHXRequest(c) {
		return render(c, pages.IndexMain())
	}
	return render(c, pages.IndexPage(u))
}

func (h *AuthHandler) PostLogin(c echo.Context) error {
	up := new(UserParams)
	if err := c.Bind(up); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid uesr data")
	}

	u, err := h.userService.GetUserByUsernameAndPassword(
		c.Request().Context(),
		up.Username,
		up.Password,
	)
	if err != nil {
		return newError(
			c, err,
			http.StatusInternalServerError,
			"invalid username or password",
		)
	}

	s, err := h.userService.CreateAuthSession(
		c.Request().Context(),
		u.UserID,
	)
	if err != nil {
		return newError(
			c, err, http.StatusInternalServerError, "unable to create session",
		)
	}

	if err := h.cookieService.SetSessionCookie(c, s.AuthSessionID); err != nil {
		return newError(
			c, err, http.StatusInternalServerError, "unable to set session cookie",
		)
	}

	if u.PasswordChangedOn == nil || u.PasswordChangedOn.IsZero() {
		return hxRedirect(c, "/auth/set-password")
	}

	return hxRedirect(c, "/app")
}

func (h *AuthHandler) GetLogOut(c echo.Context) error {
	h.cookieService.RemoveSessionCookie(c)
	return c.Redirect(http.StatusSeeOther, "/")
}

func (h *AuthHandler) GetSetPasswordPage(c echo.Context) error {
	u := getCtxUser(c)
	return render(c, pages.ChangePasswordPage(u))
}

func (h *AuthHandler) PostSetPassword(c echo.Context) error {
	up := new(UserParams)
	if err := c.Bind(up); err != nil {
		return newError(
			c, err, http.StatusBadRequest, "invalid user data",
		)
	}

	if up.Password != up.PasswordConfirm {
		return newError(
			c,
			errors.New("password != password confirm"),
			http.StatusBadRequest,
			"passwords do not match",
		)
	}

	u := getCtxUser(c)

	if up.Username != u.Username {
		return newError(c, nil, http.StatusBadRequest, "invalid username")
	}

	if err := h.userService.SetUserPassword(
		c.Request().Context(), u.UserID, up.Password,
	); err != nil {
		return newError(
			c, err, http.StatusInternalServerError, "unable to set password",
		)
	}

	h.cookieService.RemoveSessionCookie(c)
	return hxRedirect(c, "/")
}
