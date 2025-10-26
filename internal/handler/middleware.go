package handler

import (
	"net/http"

	"github.com/haatos/simple-ci/internal/service"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/labstack/echo/v4"
)

func SessionMiddleware(
	userService service.UserServicer,
	cookieService *service.CookieService,
) func(echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var user *store.User

			sessionID, err := cookieService.GetSessionID(c)
			if err == nil && sessionID != "" {
				user, err = userService.GetUserBySessionID(c.Request().Context(), sessionID)
				if err != nil {
					cookieService.RemoveSessionCookie(c)
				}
			}

			c.Set("user", user)
			return next(c)
		}
	}
}

func AlreadyLoggedIn(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := getCtxUser(c)
		if user != nil && user.PasswordChangedOn != nil && !user.PasswordChangedOn.IsZero() {
			// redirect user to main page (or somewhere else)
			return c.Redirect(http.StatusSeeOther, "/app")
		}
		return next(c)
	}
}

func IsAuthenticated(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := getCtxUser(c)
		if user == nil {
			return c.Redirect(http.StatusSeeOther, "/")
		}
		if user.PasswordChangedOn == nil || user.PasswordChangedOn.IsZero() {
			return c.Redirect(http.StatusSeeOther, "/auth/set-password")
		}
		return next(c)
	}
}

func RoleMiddleware(requiredRole store.Role) func(next echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			u := getCtxUser(c)
			if u == nil || int64(u.UserRoleID) < int64(requiredRole) {
				return newError(c, nil,
					http.StatusForbidden,
					"invalid permissions",
				)
			}
			return next(c)
		}
	}
}
