package store

import (
	"database/sql"
	"time"
)

type User struct {
	UserID            int64  `json:"-"`
	UserRoleID        Role   `json:"user_role_id"`
	Username          string `json:"username"`
	PasswordHash      string
	PasswordChangedOn *time.Time `json:"password_changed_on"`

	// session
	SessionExpires sql.NullTime `json:"session_expires"`
}

func (u *User) IsAdmin() bool {
	return u != nil && (u.UserRoleID == Admin || u.UserRoleID == Superuser)
}

func (u *User) IsSuperuser() bool {
	return u != nil && u.UserRoleID == Superuser
}

type AuthSession struct {
	AuthSessionID      string
	AuthSessionUserID  int64
	AuthSessionExpires time.Time
}
