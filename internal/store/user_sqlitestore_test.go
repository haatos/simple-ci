package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal"
	"github.com/haatos/simple-ci/internal/util"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var userStore *UserSQLiteStore
var credentialStore *CredentialSQLiteStore
var agentStore *AgentSQLiteStore
var pipelineStore *PipelineSQLiteStore
var runStore *RunSQLiteStore
var apiKeyStore *APIKeySQLiteStore

func TestMain(m *testing.M) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		log.Fatal(err)
	}

	RunMigrations(db, "../../migrations")

	userStore = NewUserSQLiteStore(db, db)
	credentialStore = NewCredentialSQLiteStore(db, db)
	agentStore = NewAgentSQLiteStore(db, db)
	pipelineStore = NewPipelineSQLiteStore(db, db)
	runStore = NewRunSQLiteStore(db, db)
	apiKeyStore = NewAPIKeySQLiteStore(db, db)
	code := m.Run()
	os.Exit(code)
}

func TestCreateUser(t *testing.T) {
	t.Run("success - user is stored", func(t *testing.T) {
		// arrange
		hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
		role := Admin
		username := "testadmin"
		sshPrivateKeyHash := string(hash)

		// act
		u, err := userStore.CreateUser(
			context.Background(),
			role,
			username,
			sshPrivateKeyHash,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, u)
		assert.NotEqual(t, 0, u.UserID)
		assert.Equal(t, role, u.UserRoleID)
		assert.Equal(t, username, u.Username)
		assert.Equal(t, sshPrivateKeyHash, u.PasswordHash)
		assert.Nil(t, u.PasswordChangedOn)
	})
	t.Run("failure - username already exists", func(t *testing.T) {
		// arrange
		hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
		role := Operator
		username := "existingoperator"
		sshPrivateKeyHash := string(hash)
		_, err := userStore.CreateUser(
			context.Background(),
			role, username, sshPrivateKeyHash,
		)
		assert.NoError(t, err)

		// act
		u, err := userStore.CreateUser(
			context.Background(),
			role, username, sshPrivateKeyHash,
		)

		// assert
		assert.Error(t, err)
		assert.Nil(t, u)
	})
}

func TestCreateSuperuser(t *testing.T) {
	t.Run("success - superuser is stored", func(t *testing.T) {
		// arrange
		hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
		username := "testsuperuser"
		sshPrivateKeyHash := string(hash)

		// act
		u, err := userStore.CreateSuperuser(
			context.Background(),
			username,
			sshPrivateKeyHash,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, u)
		assert.NotEqual(t, 0, u.UserID)
		assert.Equal(t, Superuser, u.UserRoleID)
		assert.Equal(t, username, u.Username)
		assert.Equal(t, sshPrivateKeyHash, u.PasswordHash)
		assert.NotNil(t, u.PasswordChangedOn)
		assert.True(t, u.PasswordChangedOn.Before(time.Now().UTC()))
	})
	t.Run("failure - username already exists", func(t *testing.T) {
		// arrange
		hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
		role := Superuser
		username := "existingsuperuser"
		sshPrivateKeyHash := string(hash)
		_, err := userStore.CreateUser(
			context.Background(),
			role, username, sshPrivateKeyHash,
		)
		assert.NoError(t, err)

		// act
		u, err := userStore.CreateUser(
			context.Background(),
			role, username, sshPrivateKeyHash,
		)

		// assert
		assert.Error(t, err)
		assert.Nil(t, u)
	})
}

func TestUserSQLiteStore_ReadUserByID(t *testing.T) {
	t.Run("success - user is found", func(t *testing.T) {
		// arrange
		expectedUser := createUser(t, Operator)

		// act
		u, err := userStore.ReadUserByID(context.Background(), expectedUser.UserID)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, u)
		assert.Equal(t, expectedUser.UserID, u.UserID)
		assert.Equal(t, expectedUser.UserRoleID, u.UserRoleID)
		assert.Equal(t, expectedUser.Username, u.Username)
	})
	t.Run("failure - user is not found", func(t *testing.T) {
		// arrange
		var userID int64 = 12345

		// act
		u, err := userStore.ReadUserByID(context.Background(), userID)

		// assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		assert.Nil(t, u)
	})
}

func TestUserSQLiteStore_ReadUserByUsername(t *testing.T) {
	t.Run("success - user is found", func(t *testing.T) {
		// arrange
		expectedUser := createUser(t, Operator)

		// act
		u, err := userStore.ReadUserByUsername(context.Background(), expectedUser.Username)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, u)
		assert.Equal(t, expectedUser.UserID, u.UserID)
		assert.Equal(t, expectedUser.UserRoleID, u.UserRoleID)
		assert.Equal(t, expectedUser.Username, u.Username)
	})
	t.Run("failure - user is not found", func(t *testing.T) {
		// arrange
		nonExistingUsername := "nonexistingusername"

		// act
		u, err := userStore.ReadUserByUsername(context.Background(), nonExistingUsername)

		// assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		assert.Nil(t, u)
	})
}

func TestUserSQLiteStore_DeleteUser(t *testing.T) {
	t.Run("success - user is deleted", func(t *testing.T) {
		// arrange
		expectedUser := createUser(t, Operator)

		// act
		err := userStore.DeleteUser(context.Background(), expectedUser.UserID)

		// assert
		assert.NoError(t, err)

		u, err := userStore.ReadUserByID(context.Background(), expectedUser.UserID)
		assert.Error(t, err)
		assert.Nil(t, u)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
	})
	t.Run("failure - user is not found", func(t *testing.T) {
		// arrange
		var userID int64 = 235645

		// act
		err := userStore.DeleteUser(context.Background(), userID)

		// assert
		assert.NoError(t, err)
	})
}

func TestUserSQLiteStore_CreateAuthSession(t *testing.T) {
	t.Run("success - auth session created", func(t *testing.T) {
		// arrange
		expectedUser := createUser(t, Operator)
		sessionID := "testsession"
		sessionUserID := expectedUser.UserID
		sessionExpires := time.Now().UTC().Add(30 * time.Second)

		// act
		as, err := userStore.CreateAuthSession(
			context.Background(),
			sessionID,
			sessionUserID,
			sessionExpires,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, as)
		assert.Equal(t, sessionID, as.AuthSessionID)
		assert.Equal(t, sessionUserID, as.AuthSessionUserID)
		assert.Equal(t, sessionExpires, as.AuthSessionExpires)
	})
	t.Run("failure - user id not found", func(t *testing.T) {
		// arrange
		sessionID := "testsession"
		var sessionUserID int64 = 432424
		sessionExpires := time.Now().UTC().Add(30 * time.Second)

		// act
		as, err := userStore.CreateAuthSession(
			context.Background(),
			sessionID,
			sessionUserID,
			sessionExpires,
		)

		// assert
		assert.Error(t, err)
		sqliteErr, ok := err.(*sqlite.Error)
		assert.True(t, ok)
		assert.Equal(t, sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY, sqliteErr.Code())
		assert.Nil(t, as)
	})
}

func TestUserSQLiteStore_UpdateUserRole(t *testing.T) {
	t.Run("success - user role updates", func(t *testing.T) {
		// arrange
		expectedUser := createUser(t, Operator)

		// act
		updateErr := userStore.UpdateUserRole(
			context.Background(),
			expectedUser.UserID,
			Admin,
		)
		u, readErr := userStore.ReadUserByID(context.Background(), expectedUser.UserID)

		// assert
		assert.NoError(t, updateErr)
		assert.NoError(t, readErr)
		assert.NotNil(t, u)
		assert.Equal(t, Admin, u.UserRoleID)
	})
	t.Run("failure - user role does not exist", func(t *testing.T) {
		// arrange
		expectedUser := createUser(t, Operator)

		// act
		updateErr := userStore.UpdateUserRole(
			context.Background(),
			expectedUser.UserID,
			Role(123_123_123),
		)

		// assert
		assert.Error(t, updateErr)
	})
}

func TestUserSQLiteStore_UpdateUserPassword(t *testing.T) {
	t.Run("success - user password updates", func(t *testing.T) {
		// arrange
		expectedUser := createUser(t, Operator)

		// act
		newHash, _ := bcrypt.GenerateFromPassword([]byte("newtestpassword"), bcrypt.DefaultCost)
		updateErr := userStore.UpdateUserPassword(
			context.Background(),
			expectedUser.UserID,
			string(newHash),
		)
		u, readErr := userStore.ReadUserByID(context.Background(), expectedUser.UserID)

		// assert
		assert.NoError(t, updateErr)
		assert.NoError(t, readErr)
		assert.NotNil(t, u)
		assert.Equal(t, string(newHash), u.PasswordHash)
	})
}

func TestUserSQLiteStore_UpdateUserPasswordChangedOn(t *testing.T) {
	t.Run("success - user password changed on time updates", func(t *testing.T) {
		// arrange
		expectedUser := createUser(t, Operator)

		// act
		now := time.Now().UTC()
		updateErr := userStore.UpdateUserPasswordChangedOn(
			context.Background(),
			expectedUser.UserID,
			&now,
		)
		u, readErr := userStore.ReadUserByID(context.Background(), expectedUser.UserID)

		// assert
		assert.NoError(t, updateErr)
		assert.NoError(t, readErr)
		assert.NotNil(t, u)
		assert.NotNil(t, u.PasswordChangedOn)
		assert.Equal(
			t,
			now.Format(internal.DBTimestampLayout),
			u.PasswordChangedOn.Format(internal.DBTimestampLayout),
		)
	})
}

func TestUserSQLiteStore_ReadUserBySessionID(t *testing.T) {
	t.Run("success - user is found", func(t *testing.T) {
		// arrange
		expectedUser := createUser(t, Operator)
		as := createAuthSession(t, expectedUser)

		// act
		u, err := userStore.ReadUserBySessionID(context.Background(), as.AuthSessionID)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, u)
		assert.Equal(t, expectedUser.UserID, u.UserID)
		assert.Equal(t, expectedUser.UserRoleID, u.UserRoleID)
		assert.Equal(t, expectedUser.Username, u.Username)
	})
	t.Run("failure - auth session is not found", func(t *testing.T) {
		// arrange
		nonExistingAuthSessionID := "nonexistingauthsessionid"

		// act
		u, err := userStore.ReadUserBySessionID(context.Background(), nonExistingAuthSessionID)

		// assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, sql.ErrNoRows))
		assert.Nil(t, u)
	})
}

func TestUserSQLiteStore_DeleteAuthSessionsByUserID(t *testing.T) {
	t.Run("success - user auth sessions are deleted", func(t *testing.T) {
		// arrange
		expectedUser := createUser(t, Operator)
		expectedAuthSession := createAuthSession(t, expectedUser)

		// act
		deleteErr := userStore.DeleteAuthSessionsByUserID(context.Background(), expectedUser.UserID)
		u, readErr := userStore.ReadUserBySessionID(
			context.Background(),
			expectedAuthSession.AuthSessionID,
		)

		// assert
		assert.NoError(t, deleteErr)
		assert.Error(t, readErr)
		assert.True(t, errors.Is(readErr, sql.ErrNoRows))
		assert.Nil(t, u)
	})
}

func TestUserSQLiteStore_ListUsers(t *testing.T) {
	t.Run("success - users found", func(t *testing.T) {
		// arrange
		expectedUser := createUser(t, Operator)

		// act
		users, err := userStore.ListUsers(context.Background())

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, users)
		assert.True(t, len(users) >= 1)
		assert.True(t, slices.ContainsFunc(users, func(u *User) bool {
			return u.UserID == expectedUser.UserID && u.Username == expectedUser.Username
		}))
	})
}

func TestUserSQLiteStore_ListSuperusers(t *testing.T) {
	t.Run("success - superusers found", func(t *testing.T) {
		// arrange
		expectedUser := createUser(t, Superuser)

		// act
		users, err := userStore.ListSuperusers(context.Background())

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, users)
		assert.True(t, len(users) >= 1)
		assert.True(t, slices.ContainsFunc(users, func(u User) bool {
			return u.UserID == expectedUser.UserID && u.Username == expectedUser.Username
		}))
		assert.True(t, util.All(users, func(u User) bool {
			return u.UserRoleID == Superuser
		}))
	})
}

func createUser(t *testing.T, role Role) *User {
	password := fmt.Sprintf("password%d", time.Now().UnixNano())
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	u, err := userStore.CreateUser(
		context.Background(),
		role,
		fmt.Sprintf("testuser%d", time.Now().UnixNano()),
		string(hash),
	)
	assert.NoError(t, err)
	return u
}

func createAuthSession(t *testing.T, u *User) *AuthSession {
	expectedAuthSession := &AuthSession{
		AuthSessionID:      fmt.Sprintf("%d", time.Now().UnixNano()),
		AuthSessionUserID:  u.UserID,
		AuthSessionExpires: time.Now().UTC().Add(30 * time.Second),
	}
	as, err := userStore.CreateAuthSession(
		context.Background(),
		expectedAuthSession.AuthSessionID,
		expectedAuthSession.AuthSessionUserID,
		expectedAuthSession.AuthSessionExpires,
	)
	assert.NoError(t, err)
	return as
}
