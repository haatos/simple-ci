package handler

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal/store"
	"github.com/haatos/simple-ci/internal/util"
	"github.com/haatos/simple-ci/testutil"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	_ "modernc.org/sqlite"
)

var uniqueConstraintError error

const testUserPassword string = "testpassword"

func TestMain(m *testing.M) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Exec("create table users (email text not null unique)")
	db.Exec("insert into users (email) values ('test@example.com')")
	_, uniqueConstraintError = db.Exec("insert into users (email) values ('test@example.com')")
	if uniqueConstraintError == nil {
		log.Fatal("failed to generate unique constraint error")
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestUsersHandler_ListUsers(t *testing.T) {
	t.Run("success - users found on page", func(t *testing.T) {
		// arrange
		user := generateUser(
			store.Admin,
			util.AsPtr(time.Now().UTC()),
			util.AsPtr(time.Now().UTC().Add(30*time.Second)),
		)
		expectedUsers := []*store.User{user}
		mockService := new(testutil.MockUserService)
		mockService.On("ListUsers", context.Background()).Return(expectedUsers, nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app/users", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewUserHandler(mockService, nil)

		// act
		err := h.GetUsers(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, fmt.Sprintf("<td>%s</td>", user.Username))
	})
	t.Run("success - users found on main element", func(t *testing.T) {
		// arrange
		user := generateUser(
			store.Admin,
			util.AsPtr(time.Now().UTC()),
			util.AsPtr(time.Now().UTC().Add(30*time.Second)),
		)
		expectedUsers := []*store.User{user}
		mockService := new(testutil.MockUserService)
		mockService.On("ListUsers", context.Background()).Return(expectedUsers, nil)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/app/users", nil)
		req.Header.Set("hx-request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		h := NewUserHandler(mockService, nil)

		// act
		err := h.GetUsers(c)

		// assert
		assert.NoError(t, err)
		body := rec.Body.String()
		assert.NotContains(t, body, "<!doctype html>")
		assert.Contains(t, body, "<main")
		assert.Contains(t, body, fmt.Sprintf("<td>%s</td>", user.Username))
	})
}

func TestUsersHandler_PostUsers(t *testing.T) {
	t.Run("success - user created", func(t *testing.T) {
		expectedUser := generateUser(
			store.Admin,
			util.AsPtr(time.Now().UTC()),
			util.AsPtr(time.Now().UTC().Add(30*time.Second)),
		)
		mockService := new(testutil.MockUserService)
		mockService.On(
			"CreateUser",
			context.Background(),
			expectedUser.UserRoleID,
			expectedUser.Username,
			testUserPassword,
		).Return(expectedUser, nil)

		formData := url.Values{}
		formData.Set("user_role_id", fmt.Sprintf("%d", store.Admin))
		formData.Set("username", expectedUser.Username)
		formData.Set("password", testUserPassword)
		req := httptest.NewRequest(
			http.MethodPost,
			"/app/users",
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set("hx-request", "true")
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		h := NewUserHandler(mockService, nil)

		err := h.PostUsers(c)

		assert.NoError(t, err)
		body := rec.Body.String()
		assert.Contains(t, body, fmt.Sprintf("<td>%s</td>", expectedUser.Username))
	})
	t.Run("failure - username taken", func(t *testing.T) {
		user := generateUser(store.Operator, nil, nil)
		mockService := new(testutil.MockUserService)
		mockService.On(
			"CreateUser",
			context.Background(),
			store.Admin,
			user.Username,
			testUserPassword,
		).Return(nil, uniqueConstraintError)

		formData := url.Values{}
		formData.Set("user_role_id", fmt.Sprintf("%d", store.Admin))
		formData.Set("username", user.Username)
		formData.Set("password", testUserPassword)

		req := httptest.NewRequest(
			http.MethodPost,
			"/app/users",
			bytes.NewBufferString(formData.Encode()),
		)
		req.Header.Set("hx-request", "true")
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.Request().Header.Set("hx-request", "true")
		h := NewUserHandler(mockService, nil)

		err := h.PostUsers(c)

		assert.Error(t, err)
		htmxErr, ok := err.(*HTMXError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusConflict, htmxErr.Code)
	})
}

func generateUser(
	role store.Role,
	passwordChangedOn *time.Time,
	sessionExpires *time.Time,
) *store.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(testUserPassword), bcrypt.DefaultCost)
	user := &store.User{
		UserID:            rand.Int63(),
		UserRoleID:        role,
		Username:          fmt.Sprintf("testuser%d", time.Now().UnixNano()),
		PasswordHash:      string(hash),
		PasswordChangedOn: passwordChangedOn,
	}
	if sessionExpires != nil {
		user.SessionExpires = sql.NullTime{Valid: true, Time: *sessionExpires}
	}
	return user
}
