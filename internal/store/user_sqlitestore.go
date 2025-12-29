package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/haatos/simple-ci/internal/util"

	"github.com/georgysavva/scany/v2/sqlscan"
)

type UserSQLiteStore struct {
	rdb  *sql.DB
	rwdb *sql.DB
}

func NewUserSQLiteStore(rdb, rwdb *sql.DB) *UserSQLiteStore {
	return &UserSQLiteStore{rdb, rwdb}
}

func (store *UserSQLiteStore) CreateUser(
	ctx context.Context,
	role Role,
	username string,
	passwordHash string,
) (*User, error) {
	user := new(User)
	user.UserRoleID = role
	user.Username = username
	user.PasswordHash = passwordHash
	err := sqlscan.Get(
		ctx, store.rwdb, user,
		`
		insert into users (
			user_role_id,
			username,
			password_hash
		)
		values ($1, $2, $3)
		returning user_id
		`,
		user.UserRoleID,
		user.Username,
		user.PasswordHash,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (store *UserSQLiteStore) CreateSuperuser(
	ctx context.Context,
	username string,
	passwordHash string,
) (*User, error) {
	user := new(User)
	user.UserRoleID = Superuser
	user.Username = username
	user.PasswordHash = passwordHash
	user.PasswordChangedOn = util.AsPtr(time.Now().UTC())
	err := sqlscan.Get(
		ctx, store.rwdb, user,
		`
		insert into users (
			user_role_id,
			username,
			password_hash,
			password_changed_on
		)
		values ($1, $2, $3, $4)
		returning user_id
		`,
		user.UserRoleID,
		user.Username,
		user.PasswordHash,
		user.PasswordChangedOn,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (store *UserSQLiteStore) ReadUserByID(ctx context.Context, userID int64) (*User, error) {
	user := new(User)
	user.UserID = userID
	err := sqlscan.Get(
		ctx, store.rdb, user,
		`select * from users where user_id = $1`,
		user.UserID,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (store *UserSQLiteStore) ReadUserByUsername(
	ctx context.Context,
	username string,
) (*User, error) {
	user := new(User)
	user.Username = username
	err := sqlscan.Get(
		ctx, store.rdb, user,
		`select * from users where username = $1`,
		user.Username,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (store *UserSQLiteStore) ReadUserBySessionID(
	ctx context.Context,
	sessionID string,
) (*User, error) {
	user := &User{}
	err := sqlscan.Get(
		ctx, store.rdb, user,
		`select
			u.user_id,
			u.user_role_id,
			u.username,
			u.password_hash,
			u.password_changed_on,
			s.auth_session_expires as session_expires
		from users u
		left join auth_sessions s
		on u.user_id = s.auth_session_user_id
		where s.auth_session_id = $1
		order by s.auth_session_expires DESC
		limit 1
		`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (store *UserSQLiteStore) DeleteUser(ctx context.Context, userID int64) error {
	_, err := store.rwdb.ExecContext(ctx, "delete from users where user_id = $1", userID)
	return err
}

func (store *UserSQLiteStore) UpdateUserRole(
	ctx context.Context,
	userID int64,
	role Role,
) error {
	_, err := store.rwdb.ExecContext(
		ctx,
		`update users
		set user_role_id = $1
		where user_id = $2`,
		role, userID)
	return err
}

func (store *UserSQLiteStore) UpdateUserPassword(
	ctx context.Context,
	userID int64,
	passwordHash string,
	changedOn *time.Time,
) error {
	query := `update users
	set password_hash = $1,
		password_changed_on = $2
	where user_id = $3`
	_, err := store.rwdb.ExecContext(ctx, query, passwordHash, changedOn, userID)
	return err
}

func (store *UserSQLiteStore) CreateAuthSession(
	ctx context.Context,
	authSessionID string,
	userID int64,
	expires time.Time,
) (*AuthSession, error) {
	as := &AuthSession{
		AuthSessionID:      authSessionID,
		AuthSessionUserID:  userID,
		AuthSessionExpires: expires,
	}
	_, err := store.rwdb.ExecContext(
		ctx,
		`
		insert into auth_sessions (
			auth_session_id,
			auth_session_user_id,
			auth_session_expires
		)
		values ($1, $2, $3)
		`,
		as.AuthSessionID,
		as.AuthSessionUserID,
		as.AuthSessionExpires,
	)
	if err != nil {
		return nil, err
	}
	return as, nil
}

func (store *UserSQLiteStore) DeleteAuthSessionsByUserID(ctx context.Context, userID int64) error {
	_, err := store.rwdb.ExecContext(
		ctx,
		`delete from auth_sessions
		where auth_session_user_id = $1`,
		userID,
	)
	return err
}

func (store *UserSQLiteStore) ListUsers(ctx context.Context) ([]*User, error) {
	var users []*User
	err := sqlscan.Select(
		ctx, store.rdb, &users,
		"select * from users order by username",
	)
	return users, err
}

func (store *UserSQLiteStore) ListSuperusers(ctx context.Context) ([]User, error) {
	users := make([]User, 0)
	query := `select * from users where user_role_id = 10000`
	err := sqlscan.Select(ctx, store.rdb, &users, query)
	return users, err
}
