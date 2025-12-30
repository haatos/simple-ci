package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/views"
	"github.com/haatos/simple-ci/internal/views/pages"

	"github.com/labstack/echo/v4"
)

type UserCookieServicer interface {
	RemoveSessionCookie(echo.Context)
}

func SetupUserRoutes(
	g *echo.Group,
	userService UserServicer,
	cookieService UserCookieServicer,
) {
	h := NewUserHandler(userService, cookieService)
	usersGroup := g.Group("/app/users", IsAuthenticated)
	usersGroup.GET("", h.GetUsers, RoleMiddleware(store.Admin))
	usersGroup.POST("", h.PostUsers, RoleMiddleware(store.Admin))
	usersGroup.GET("/profile", h.GetProfilePage)
	usersGroup.DELETE("/:user_id", h.DeleteUser, RoleMiddleware(store.Admin))
	usersGroup.PATCH("/:user_id/change-password", h.PatchChangeUserPassword)
	usersGroup.PATCH("/:user_id/password", h.PatchUserPassword)
	usersGroup.PATCH(
		"/:user_id/reset-password",
		h.PatchResetUserPassword,
		RoleMiddleware(store.Admin),
	)
	usersGroup.PATCH("/:user_id/role", h.PatchUserRole, RoleMiddleware(store.Superuser))
}

type UserWriter interface {
	CreateUser(
		ctx context.Context,
		userRoleID store.Role,
		username, password string,
	) (*store.User, error)
	ChangeUserPassword(
		ctx context.Context,
		userID int64,
		oldPassword, newPassword string,
	) error
	ResetUserPassword(
		ctx context.Context,
		userID int64,
		newPassword string,
	) error
	DeleteUser(ctx context.Context, u *store.User) error
	UpdateUserRole(ctx context.Context, userID int64, role store.Role) error
	InitializeSuperuser(context.Context)
}

type UserReader interface {
	GetUserByID(ctx context.Context, userID int64) (*store.User, error)
	GetUserBySessionID(ctx context.Context, sessionID string) (*store.User, error)
	ListUsers(ctx context.Context) ([]*store.User, error)
	ListSuperusers(ctx context.Context) ([]store.User, error)
}

type UserServicer interface {
	UserWriter
	UserReader
}

type UserHandler struct {
	userService   UserServicer
	cookieService UserCookieServicer
}

func NewUserHandler(
	userService UserServicer,
	cookieService UserCookieServicer,
) *UserHandler {
	return &UserHandler{userService, cookieService}
}

const MaxUsersPerPage = 20

func (h *UserHandler) GetUsers(c echo.Context) error {
	u := getCtxUser(c)
	users, err := h.userService.ListUsers(c.Request().Context())
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusInternalServerError, "unable to list users")
		}
	}

	if isHXRequest(c) {
		return render(c, pages.UsersPageMain(users))
	}
	return render(c, pages.UsersPage(u, users))
}

func (h *UserHandler) PostUsers(c echo.Context) error {
	up := new(UserParams)
	if err := c.Bind(up); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid user data")
	}

	u, err := h.userService.CreateUser(
		c.Request().Context(),
		up.UserRoleID,
		up.Username,
		up.Password,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return newError(
				c, err,
				http.StatusConflict,
				fmt.Sprintf("A user with username '%s' already exists", up.Username),
			)
		}
		return newError(c, err, http.StatusInternalServerError, "Unable to create user")
	}

	return render(c, pages.UserRow(u))
}

func (h *UserHandler) PatchChangeUserPassword(c echo.Context) error {
	ctxUser := getCtxUser(c)

	pup := new(PatchUserPasswordParams)
	if err := c.Bind(pup); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid user data")
	}
	if pup.Password != pup.PasswordConfirm {
		return newError(c, nil, http.StatusBadRequest, "passwords do not match")
	}

	if pup.UserID != ctxUser.UserID {
		return newError(c, nil, http.StatusForbidden, "unable to change another user's password")
	}

	if err := h.userService.ChangeUserPassword(
		c.Request().Context(),
		pup.UserID,
		pup.OldPassword,
		pup.Password,
	); err != nil {
		return newError(c, err, http.StatusInternalServerError, "unable to change user's password")
	}

	h.cookieService.RemoveSessionCookie(c)
	return hxRedirect(c, "/")
}

func (h *UserHandler) PatchUserPassword(c echo.Context) error {
	pup := new(PatchUserPasswordParams)
	if err := c.Bind(pup); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid user data")
	}
	if pup.Password != pup.PasswordConfirm {
		return newError(c, nil, http.StatusBadRequest, "passwords do not match")
	}

	u, err := h.userService.GetUserByID(c.Request().Context(), pup.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusBadRequest, "user not found")
		}
		return newError(c, err, http.StatusInternalServerError, "something went wrong")
	}

	if err := h.userService.ChangeUserPassword(
		c.Request().Context(),
		pup.UserID,
		pup.OldPassword,
		pup.Password,
	); err != nil {
		return newError(c, err, http.StatusInternalServerError, "unable to update user's password")
	}

	return renderToast(
		c,
		views.SuccessToast(fmt.Sprintf("User '%s' password updated", u.Username), 3000),
	)
}

func (h *UserHandler) PatchResetUserPassword(c echo.Context) error {
	pup := new(PatchUserPasswordParams)
	if err := c.Bind(pup); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid user data")
	}

	u, err := h.userService.GetUserByID(c.Request().Context(), pup.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusBadRequest, "user not found")
		}
		return newError(c, err, http.StatusInternalServerError, "something went wrong")
	}

	if err := h.userService.ResetUserPassword(
		c.Request().Context(), pup.UserID, pup.Password,
	); err != nil {
		return newError(c, err, http.StatusInternalServerError, "unable to reset user's password")
	}

	return renderToast(
		c,
		views.SuccessToast(fmt.Sprintf("Password reset for '%s'", u.Username), 3000),
	)
}

func (h *UserHandler) DeleteUser(c echo.Context) error {
	pup := new(PatchUserParams)
	if err := c.Bind(pup); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid user data")
	}

	user, err := h.userService.GetUserByID(c.Request().Context(), pup.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusBadRequest, "user not found")
		}
		return newError(c, err, http.StatusInternalServerError, "something went wrong")
	}

	if user.IsSuperuser() {
		return newError(c, err, http.StatusForbidden, "cannot delete superuser")
	}

	if err := h.userService.DeleteUser(c.Request().Context(), user); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusBadRequest, "user not found")
		}
		return newError(c, err, http.StatusInternalServerError, "unable to delete user.")
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *UserHandler) PatchUserRole(c echo.Context) error {
	pu := new(PatchUserParams)
	if err := c.Bind(pu); err != nil {
		return newError(c, err, http.StatusBadRequest, "invalid user data")
	}

	u, err := h.userService.GetUserByID(c.Request().Context(), pu.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return newError(c, err, http.StatusBadRequest, "user not found")
		}
		return newError(c, err, http.StatusInternalServerError, "something went wrong")
	}

	if err := h.userService.UpdateUserRole(
		c.Request().Context(),
		pu.UserID,
		pu.RoleID,
	); err != nil {
		return newError(c, err, http.StatusInternalServerError, "unable to update user role")
	}

	return renderToast(c, views.SuccessToast(
		fmt.Sprintf(
			"user '%s' role updated to '%s'",
			u.Username, pu.RoleID.ToString(),
		), 3000))
}

func (h *UserHandler) GetProfilePage(c echo.Context) error {
	u := getCtxUser(c)

	if isHXRequest(c) {
		return render(c, pages.ProfileMain(u))
	}
	return render(c, pages.ProfilePage(u))
}
