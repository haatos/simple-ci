package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/haatos/simple-ci/internal/types"
)

type User struct {
	UserID            int64      `json:"-"`
	UserRoleID        types.Role `json:"user_role_id"`
	Username          string     `json:"username"`
	PasswordHash      string
	PasswordChangedOn *time.Time `json:"password_changed_on"`

	// session
	SessionExpires sql.NullTime `json:"session_expires"`
}

func (u *User) IsAdmin() bool {
	return u != nil && (u.UserRoleID == types.Admin || u.UserRoleID == types.Superuser)
}

func (u *User) IsSuperuser() bool {
	return u != nil && u.UserRoleID == types.Superuser
}

type AuthSession struct {
	AuthSessionID      string
	AuthSessionUserID  int64
	AuthSessionExpires time.Time
}

type UserStore interface {
	CreateUser(context.Context, types.Role, string, string) (*User, error)
	ReadUserByID(context.Context, int64) (*User, error)
	ReadUserByUsername(context.Context, string) (*User, error)
	ReadUserBySessionID(context.Context, string) (*User, error)
	UpdateUserRole(context.Context, int64, types.Role) error
	UpdateUserPassword(context.Context, int64, string) error
	UpdateUserPasswordChangedOn(context.Context, int64, *time.Time) error
	DeleteUser(context.Context, int64) error
	ListUsers(context.Context) ([]*User, error)
	ListSuperusers(context.Context) ([]User, error)

	CreateAuthSession(context.Context, string, int64, time.Time) (*AuthSession, error)
	DeleteAuthSessionsByUserID(context.Context, int64) error
}
