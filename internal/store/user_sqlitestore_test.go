package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"slices"
	"testing"
	"time"

	"github.com/haatos/simple-ci/internal/util"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type userSQLiteStoreSuite struct {
	userStore *UserSQLiteStore
	db        *sql.DB
	suite.Suite
}

func TestUserSQLiteStore(t *testing.T) {
	suite.Run(t, new(userSQLiteStoreSuite))
}

func (suite *userSQLiteStoreSuite) SetupSuite() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	suite.db = db
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		log.Fatal(err)
	}

	RunMigrations(db, "../../migrations")

	suite.userStore = NewUserSQLiteStore(db, db)
}

func (suite *userSQLiteStoreSuite) TearDownSuite() {
	_ = suite.db.Close()
}

func (suite *userSQLiteStoreSuite) TestUserSQLiteStore_CreateUser() {
	suite.Run("success - user is stored", func() {
		// arrange
		hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
		role := Admin
		username := "testadmin"
		sshPrivateKeyHash := string(hash)

		// act
		u, err := suite.userStore.CreateUser(
			context.Background(),
			role,
			username,
			sshPrivateKeyHash,
		)

		// assert
		suite.NoError(err)
		suite.NotNil(u)
		suite.NotEqual(0, u.UserID)
		suite.Equal(role, u.UserRoleID)
		suite.Equal(username, u.Username)
		suite.Equal(sshPrivateKeyHash, u.PasswordHash)
		suite.Nil(u.PasswordChangedOn)
	})
	suite.Run("failure - username already exists", func() {
		// arrange
		hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
		role := Operator
		username := "existingoperator"
		sshPrivateKeyHash := string(hash)
		_, err := suite.userStore.CreateUser(
			context.Background(),
			role, username, sshPrivateKeyHash,
		)
		suite.NoError(err)

		// act
		u, err := suite.userStore.CreateUser(
			context.Background(),
			role, username, sshPrivateKeyHash,
		)

		// assert
		suite.Error(err)
		suite.Nil(u)
	})
}

func (suite *userSQLiteStoreSuite) TestUserSQLiteStore_CreateSuperuser() {
	suite.Run("success - superuser is stored", func() {
		// arrange
		hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
		username := "testsuperuser"
		sshPrivateKeyHash := string(hash)

		// act
		u, err := suite.userStore.CreateSuperuser(
			context.Background(),
			username,
			sshPrivateKeyHash,
		)

		// assert
		suite.NoError(err)
		suite.NotNil(u)
		suite.NotEqual(0, u.UserID)
		suite.Equal(Superuser, u.UserRoleID)
		suite.Equal(username, u.Username)
		suite.Equal(sshPrivateKeyHash, u.PasswordHash)
		suite.NotNil(u.PasswordChangedOn)
		suite.True(u.PasswordChangedOn.Before(time.Now().UTC()))
	})
	suite.Run("failure - username already exists", func() {
		// arrange
		hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
		role := Superuser
		username := "existingsuperuser"
		sshPrivateKeyHash := string(hash)
		_, err := suite.userStore.CreateUser(
			context.Background(),
			role, username, sshPrivateKeyHash,
		)
		suite.NoError(err)

		// act
		u, err := suite.userStore.CreateUser(
			context.Background(),
			role, username, sshPrivateKeyHash,
		)

		// assert
		suite.Error(err)
		suite.Nil(u)
	})
}

func (suite *userSQLiteStoreSuite) TestUserSQLiteStore_ReadUserByID() {
	suite.Run("success - user is found", func() {
		// arrange
		expectedUser := suite.createUser(Operator)

		// act
		u, err := suite.userStore.ReadUserByID(context.Background(), expectedUser.UserID)

		// assert
		suite.NoError(err)
		suite.NotNil(u)
		suite.Equal(expectedUser.UserID, u.UserID)
		suite.Equal(expectedUser.UserRoleID, u.UserRoleID)
		suite.Equal(expectedUser.Username, u.Username)
	})
	suite.Run("failure - user is not found", func() {
		// arrange
		var userID int64 = 12345

		// act
		u, err := suite.userStore.ReadUserByID(context.Background(), userID)

		// assert
		suite.Error(err)
		suite.True(errors.Is(err, sql.ErrNoRows))
		suite.Nil(u)
	})
}

func (suite *userSQLiteStoreSuite) TestUserSQLiteStore_ReadUserByUsername() {
	suite.Run("success - user is found", func() {
		// arrange
		expectedUser := suite.createUser(Operator)

		// act
		u, err := suite.userStore.ReadUserByUsername(context.Background(), expectedUser.Username)

		// assert
		suite.NoError(err)
		suite.NotNil(u)
		suite.Equal(expectedUser.UserID, u.UserID)
		suite.Equal(expectedUser.UserRoleID, u.UserRoleID)
		suite.Equal(expectedUser.Username, u.Username)
	})
	suite.Run("failure - user is not found", func() {
		// arrange
		nonExistingUsername := "nonexistingusername"

		// act
		u, err := suite.userStore.ReadUserByUsername(context.Background(), nonExistingUsername)

		// assert
		suite.Error(err)
		suite.True(errors.Is(err, sql.ErrNoRows))
		suite.Nil(u)
	})
}

func (suite *userSQLiteStoreSuite) TestUserSQLiteStore_DeleteUser() {
	suite.Run("success - user is deleted", func() {
		// arrange
		expectedUser := suite.createUser(Operator)

		// act
		err := suite.userStore.DeleteUser(context.Background(), expectedUser.UserID)

		// assert
		suite.NoError(err)

		u, err := suite.userStore.ReadUserByID(context.Background(), expectedUser.UserID)
		suite.Error(err)
		suite.Nil(u)
		suite.True(errors.Is(err, sql.ErrNoRows))
	})
	suite.Run("failure - user is not found", func() {
		// arrange
		var userID int64 = 235645

		// act
		err := suite.userStore.DeleteUser(context.Background(), userID)

		// assert
		suite.NoError(err)
	})
}

func (suite *userSQLiteStoreSuite) TestUserSQLiteStore_CreateAuthSession() {
	suite.Run("success - auth session created", func() {
		// arrange
		expectedUser := suite.createUser(Operator)
		sessionID := "testsession"
		sessionUserID := expectedUser.UserID
		sessionExpires := time.Now().UTC().Add(30 * time.Second)

		// act
		as, err := suite.userStore.CreateAuthSession(
			context.Background(),
			sessionID,
			sessionUserID,
			sessionExpires,
		)

		// assert
		suite.NoError(err)
		suite.NotNil(as)
		suite.Equal(sessionID, as.AuthSessionID)
		suite.Equal(sessionUserID, as.AuthSessionUserID)
		suite.Equal(sessionExpires, as.AuthSessionExpires)
	})
	suite.Run("failure - user id not found", func() {
		// arrange
		sessionID := "testsession"
		var sessionUserID int64 = 432424
		sessionExpires := time.Now().UTC().Add(30 * time.Second)

		// act
		as, err := suite.userStore.CreateAuthSession(
			context.Background(),
			sessionID,
			sessionUserID,
			sessionExpires,
		)

		// assert
		suite.Error(err)
		sqliteErr, ok := err.(*sqlite.Error)
		suite.True(ok)
		suite.Equal(sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY, sqliteErr.Code())
		suite.Nil(as)
	})
}

func (suite *userSQLiteStoreSuite) TestUserSQLiteStore_UpdateUserRole() {
	suite.Run("success - user role updates", func() {
		// arrange
		expectedUser := suite.createUser(Operator)

		// act
		updateErr := suite.userStore.UpdateUserRole(
			context.Background(),
			expectedUser.UserID,
			Admin,
		)
		u, readErr := suite.userStore.ReadUserByID(context.Background(), expectedUser.UserID)

		// assert
		suite.NoError(updateErr)
		suite.NoError(readErr)
		suite.NotNil(u)
		suite.Equal(Admin, u.UserRoleID)
	})
	suite.Run("failure - user role does not exist", func() {
		// arrange
		expectedUser := suite.createUser(Operator)

		// act
		updateErr := suite.userStore.UpdateUserRole(
			context.Background(),
			expectedUser.UserID,
			Role(123_123_123),
		)

		// assert
		suite.Error(updateErr)
	})
}

func (suite *userSQLiteStoreSuite) TestUserSQLiteStore_UpdateUserPassword() {
	suite.Run("success - user password updates", func() {
		// arrange
		expectedUser := suite.createUser(Operator)

		// act
		newHash, _ := bcrypt.GenerateFromPassword([]byte("newtestpassword"), bcrypt.DefaultCost)
		now := time.Now().UTC()
		updateErr := suite.userStore.UpdateUserPassword(
			context.Background(),
			expectedUser.UserID,
			string(newHash),
			&now,
		)
		u, readErr := suite.userStore.ReadUserByID(context.Background(), expectedUser.UserID)

		// assert
		suite.NoError(updateErr)
		suite.NoError(readErr)
		suite.NotNil(u)
		suite.Equal(string(newHash), u.PasswordHash)
	})
}

func (suite *userSQLiteStoreSuite) TestUserSQLiteStore_ReadUserBySessionID() {
	suite.Run("success - user is found", func() {
		// arrange
		expectedUser := suite.createUser(Operator)
		as := suite.createAuthSession(expectedUser)

		// act
		u, err := suite.userStore.ReadUserBySessionID(context.Background(), as.AuthSessionID)

		// assert
		suite.NoError(err)
		suite.NotNil(u)
		suite.Equal(expectedUser.UserID, u.UserID)
		suite.Equal(expectedUser.UserRoleID, u.UserRoleID)
		suite.Equal(expectedUser.Username, u.Username)
	})
	suite.Run("failure - auth session is not found", func() {
		// arrange
		nonExistingAuthSessionID := "nonexistingauthsessionid"

		// act
		u, err := suite.userStore.ReadUserBySessionID(
			context.Background(),
			nonExistingAuthSessionID,
		)

		// assert
		suite.Error(err)
		suite.True(errors.Is(err, sql.ErrNoRows))
		suite.Nil(u)
	})
}

func (suite *userSQLiteStoreSuite) TestUserSQLiteStore_DeleteAuthSessionsByUserID() {
	suite.Run("success - user auth sessions are deleted", func() {
		// arrange
		expectedUser := suite.createUser(Operator)
		expectedAuthSession := suite.createAuthSession(expectedUser)

		// act
		deleteErr := suite.userStore.DeleteAuthSessionsByUserID(
			context.Background(),
			expectedUser.UserID,
		)
		u, readErr := suite.userStore.ReadUserBySessionID(
			context.Background(),
			expectedAuthSession.AuthSessionID,
		)

		// assert
		suite.NoError(deleteErr)
		suite.Error(readErr)
		suite.True(errors.Is(readErr, sql.ErrNoRows))
		suite.Nil(u)
	})
}

func (suite *userSQLiteStoreSuite) TestUserSQLiteStore_ListUsers() {
	suite.Run("success - users found", func() {
		// arrange
		expectedUser := suite.createUser(Operator)

		// act
		users, err := suite.userStore.ListUsers(context.Background())

		// assert
		suite.NoError(err)
		suite.NotNil(users)
		suite.True(len(users) >= 1)
		suite.True(slices.ContainsFunc(users, func(u *User) bool {
			return u.UserID == expectedUser.UserID && u.Username == expectedUser.Username
		}))
	})
}

func (suite *userSQLiteStoreSuite) TestUserSQLiteStore_ListSuperusers() {
	suite.Run("success - superusers found", func() {
		// arrange
		expectedUser := suite.createUser(Superuser)

		// act
		users, err := suite.userStore.ListSuperusers(context.Background())

		// assert
		suite.NoError(err)
		suite.NotNil(users)
		suite.True(len(users) >= 1)
		suite.True(slices.ContainsFunc(users, func(u User) bool {
			return u.UserID == expectedUser.UserID && u.Username == expectedUser.Username
		}))
		suite.True(util.All(users, func(u User) bool {
			return u.UserRoleID == Superuser
		}))
	})
}

func (suite *userSQLiteStoreSuite) createUser(role Role) *User {
	password := fmt.Sprintf("password%d", time.Now().UTC().UnixNano())
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	u, err := suite.userStore.CreateUser(
		context.Background(),
		role,
		fmt.Sprintf("testuser%d", time.Now().UTC().UnixNano()),
		string(hash),
	)
	suite.NoError(err)
	return u
}

func (suite *userSQLiteStoreSuite) createAuthSession(u *User) *AuthSession {
	expectedAuthSession := &AuthSession{
		AuthSessionID:      fmt.Sprintf("%d", time.Now().UTC().UnixNano()),
		AuthSessionUserID:  u.UserID,
		AuthSessionExpires: time.Now().UTC().Add(30 * time.Second),
	}
	as, err := suite.userStore.CreateAuthSession(
		context.Background(),
		expectedAuthSession.AuthSessionID,
		expectedAuthSession.AuthSessionUserID,
		expectedAuthSession.AuthSessionExpires,
	)
	suite.NoError(err)
	return as
}
