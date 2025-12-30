package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/haatos/simple-ci/internal/settings"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/util"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

type UserWriter interface {
	CreateUser(context.Context, store.Role, string, string) (*store.User, error)
	CreateSuperuser(context.Context, string, string) (*store.User, error)
	UpdateUserRole(context.Context, int64, store.Role) error
	UpdateUserPassword(context.Context, int64, string, *time.Time) error
	DeleteUser(context.Context, int64) error
}

type UserReader interface {
	ReadUserByID(context.Context, int64) (*store.User, error)
	ReadUserByUsername(context.Context, string) (*store.User, error)
	ReadUserBySessionID(context.Context, string) (*store.User, error)
	ListUsers(context.Context) ([]*store.User, error)
	ListSuperusers(context.Context) ([]store.User, error)
}

type AuthSessionWriter interface {
	CreateAuthSession(context.Context, string, int64, time.Time) (*store.AuthSession, error)
	DeleteAuthSessionsByUserID(context.Context, int64) error
}

type UserStore interface {
	UserWriter
	UserReader
	AuthSessionWriter
}

type UserService struct {
	userStore UserStore
}

func NewUserService(s UserStore) *UserService {
	return &UserService{userStore: s}
}

func (s *UserService) GetUserByID(ctx context.Context, userID int64) (*store.User, error) {
	return s.userStore.ReadUserByID(ctx, userID)
}

func (s *UserService) GetUserBySessionID(
	ctx context.Context,
	sessionID string,
) (*store.User, error) {
	u, err := s.userStore.ReadUserBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !u.SessionExpires.Valid || u.SessionExpires.Time.Before(time.Now().UTC()) {
		return nil, errors.New("session expired")
	}
	return u, nil
}

func (s *UserService) CreateAuthSession(
	ctx context.Context,
	userID int64,
) (*store.AuthSession, error) {
	as, err := s.userStore.CreateAuthSession(
		ctx,
		generateRandomSessionID(),
		userID,
		time.Now().UTC().Add(settings.Settings.SessionExpires),
	)
	return as, err
}

func generateRandomSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *UserService) GetUserByUsernameAndPassword(
	ctx context.Context,
	username, password string,
) (*store.User, error) {
	u, err := s.userStore.ReadUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) CreateUser(
	ctx context.Context,
	userRoleID store.Role,
	username, password string,
) (*store.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u, err := s.userStore.CreateUser(ctx, userRoleID, username, string(hash))
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) ListUsers(ctx context.Context) ([]*store.User, error) {
	users, err := s.userStore.ListUsers(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return users, nil
}

func (s *UserService) ChangeUserPassword(
	ctx context.Context,
	userID int64,
	oldPassword, newPassword string,
) error {
	u, err := s.userStore.ReadUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.IsSuperuser() {
		return errors.New("attempt to change superuser's password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)); err != nil {
		return err
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(newHash)
	u.PasswordChangedOn = util.AsPtr(time.Now().UTC())
	return s.userStore.UpdateUserPassword(ctx, u.UserID, u.PasswordHash, u.PasswordChangedOn)
}

func (s *UserService) SetUserPassword(
	ctx context.Context,
	userID int64,
	newPassword string,
) error {
	u, err := s.userStore.ReadUserByID(ctx, userID)
	if err != nil {
		return err
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(newHash)
	u.PasswordChangedOn = util.AsPtr(time.Now().UTC())
	return s.userStore.UpdateUserPassword(ctx, u.UserID, u.PasswordHash, u.PasswordChangedOn)
}

func (s *UserService) ResetUserPassword(
	ctx context.Context,
	userID int64,
	newPassword string,
) error {
	u, err := s.userStore.ReadUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.IsSuperuser() {
		return errors.New("attempt to reset superuser's password")
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(newHash)
	u.PasswordChangedOn = nil
	return s.userStore.UpdateUserPassword(ctx, u.UserID, u.PasswordHash, u.PasswordChangedOn)
}

func (s *UserService) DeleteUser(ctx context.Context, u *store.User) error {
	return s.userStore.DeleteUser(ctx, u.UserID)
}

func (s *UserService) ListSuperusers(ctx context.Context) ([]store.User, error) {
	users, err := s.userStore.ListSuperusers(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return users, nil
}

func (s *UserService) UpdateUserRole(ctx context.Context, userID int64, role store.Role) error {
	return s.userStore.UpdateUserRole(ctx, userID, role)
}

func (s *UserService) InitializeSuperuser(ctx context.Context) {
	users, err := s.ListSuperusers(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Fatal(err)
	}
	if len(users) == 0 {
		fmt.Println("Create a superuser")
		fmt.Print("Username: ")
		var username string
		_, err := fmt.Scanln(&username)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatal(err)
		}

		hash, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)
		if err != nil {
			log.Fatal(err)
		}

		if _, err = s.userStore.CreateSuperuser(
			ctx,
			username,
			string(hash),
		); err != nil {
			log.Fatal(err)
		}
	}
}
