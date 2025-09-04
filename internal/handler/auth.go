package handler

import (
	"errors"
	"net/http"

	"github.com/haatos/simple-ci/internal/service"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/views/pages"

	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	userService   service.UserServicer
	cookieService *service.CookieService
}

func NewAuthHandler(
	userService service.UserServicer,
	cookieService *service.CookieService,
) *AuthHandler {
	return &AuthHandler{userService, cookieService}
}

func (h *AuthHandler) SessionMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var user *store.User

		sessionID, err := h.cookieService.GetSessionID(c)
		if err == nil && sessionID != "" {
			user, err = h.userService.GetUserBySessionID(c.Request().Context(), sessionID)
			if err != nil {
				h.cookieService.RemoveSessionCookie(c)
			}
		}

		c.Set("user", user)
		return next(c)
	}
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
		return newError(c, err, http.StatusInternalServerError, "invalid username or password")
	}

	s, err := h.userService.CreateAuthSession(
		c.Request().Context(),
		u.UserID,
	)
	if err != nil {
		return newError(c, err, http.StatusInternalServerError, "unable to create session")
	}

	if err := h.cookieService.SetSessionCookie(c, s.AuthSessionID); err != nil {
		return newError(c, err, http.StatusInternalServerError, "unable to set session cookie")
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

	if err := h.userService.SetUserPassword(c.Request().Context(), u.UserID, up.Password); err != nil {
		return newError(
			c, err, http.StatusInternalServerError, "unable to set password",
		)
	}

	h.cookieService.RemoveSessionCookie(c)
	return hxRedirect(c, "/")
}
