package service

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/haatos/simple-ci/internal/security"
	"github.com/haatos/simple-ci/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCredentialStore struct {
	mock.Mock
}

func (m *MockCredentialStore) CreateCredential(
	ctx context.Context,
	username,
	description,
	sshPrivateKey string,
) (*store.Credential, error) {
	args := m.Called(ctx, username, description, sshPrivateKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Credential), args.Error(1)
}

func (m *MockCredentialStore) ReadCredentialByID(
	ctx context.Context,
	id int64,
) (*store.Credential, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Credential), args.Error(1)
}

func (m *MockCredentialStore) UpdateCredential(
	ctx context.Context,
	id int64,
	username, description string,
) error {
	args := m.Called(ctx, id, username, description)
	return args.Error(0)
}

func (m *MockCredentialStore) DeleteCredential(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCredentialStore) ListCredentials(ctx context.Context) ([]*store.Credential, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*store.Credential), args.Error(1)
}

type MockEncrypter struct {
	mock.Mock
}

func (m *MockEncrypter) EncryptAES(text string) string {
	args := m.Called(text)
	return args.Get(0).(string)
}

func (m *MockEncrypter) DecryptAES(hash string) ([]byte, error) {
	args := m.Called(hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), nil
}

func TestCredentialService_CreateCredential(t *testing.T) {
	t.Run("success - credential created", func(t *testing.T) {
		// arrange
		mockEncrypter, testPrivateKey, expectedCredential := generateCredential()
		mockStore := new(MockCredentialStore)
		mockStore.On(
			"CreateCredential",
			context.Background(),
			expectedCredential.Username,
			expectedCredential.Description,
			expectedCredential.SSHPrivateKeyHash,
		).Return(expectedCredential, nil)
		credentialService := NewCredentialService(mockStore, mockEncrypter)

		// act
		credential, err := credentialService.CreateCredential(
			context.Background(),
			expectedCredential.Username,
			expectedCredential.Description,
			testPrivateKey,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, credential)
		assert.Equal(t, expectedCredential.Username, credential.Username)
		assert.Equal(t, expectedCredential.Description, credential.Description)
		assert.Equal(t, expectedCredential.SSHPrivateKeyHash, credential.SSHPrivateKeyHash)
	})
}

func TestCredentialService_GetCredentialByID(t *testing.T) {
	t.Run("success - credential is found", func(t *testing.T) {
		// arrange
		mockEncrypter, _, expectedCredential := generateCredential()
		mockStore := new(MockCredentialStore)
		mockStore.On(
			"ReadCredentialByID",
			context.Background(),
			expectedCredential.CredentialID,
		).Return(expectedCredential, nil)
		credentialService := NewCredentialService(mockStore, mockEncrypter)

		// act
		credential, err := credentialService.GetCredentialByID(
			context.Background(),
			expectedCredential.CredentialID,
		)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, credential)
	})
}

func TestCredentialService_UpdateCredential(t *testing.T) {
	t.Run("success - credential updated", func(t *testing.T) {
		// arrange
		mockEncrypter, _, expectedCredential := generateCredential()
		mockStore := new(MockCredentialStore)
		mockStore.On(
			"UpdateCredential",
			context.Background(),
			expectedCredential.CredentialID,
			expectedCredential.Username,
			expectedCredential.Description,
		).Return(nil)
		credentialService := NewCredentialService(mockStore, mockEncrypter)

		// act
		err := credentialService.UpdateCredential(
			context.Background(),
			expectedCredential.CredentialID,
			expectedCredential.Username,
			expectedCredential.Description,
		)

		// assert
		assert.NoError(t, err)
	})
}

func TestCredentialService_DeleteCredential(t *testing.T) {
	t.Run("success - credential found", func(t *testing.T) {
		// arrange
		mockEncrypter, _, expectedCredential := generateCredential()
		mockStore := new(MockCredentialStore)
		mockStore.On(
			"DeleteCredential",
			context.Background(),
			expectedCredential.CredentialID,
		).Return(nil)
		credentialService := NewCredentialService(mockStore, mockEncrypter)

		// act
		err := credentialService.DeleteCredential(
			context.Background(),
			expectedCredential.CredentialID,
		)

		// assert
		assert.NoError(t, err)
	})
}

func TestCredentialService_ListCredentials(t *testing.T) {
	t.Run("success - credentials found", func(t *testing.T) {
		// arrange
		mockEncrypter, _, expectedCredential := generateCredential()
		mockStore := new(MockCredentialStore)
		expectedCredentials := []*store.Credential{expectedCredential}
		mockStore.On(
			"ListCredentials", context.Background(),
		).Return(expectedCredentials, nil)
		credentialService := NewCredentialService(mockStore, mockEncrypter)

		// act
		credentials, err := credentialService.ListCredentials(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, len(expectedCredentials), len(credentials))
	})
}

func newMockEncrypter() (*MockEncrypter, string, string) {
	mockEncrypter := new(MockEncrypter)
	testPrivateKey := "testprivatekey"
	testPrivateKeyHash := "testprivatekeyhash"
	mockEncrypter.On("EncryptAES", testPrivateKey).Return(testPrivateKeyHash)
	mockEncrypter.On("DecryptAES", testPrivateKeyHash).Return([]byte(testPrivateKey), nil)
	return mockEncrypter, testPrivateKey, testPrivateKeyHash
}

func generateCredential() (*MockEncrypter, string, *store.Credential) {
	os.Setenv("SIMPLECI_HASH_KEY", security.GenerateRandomKey(32))
	mockEncrypter, testPrivateKey, testPrivateKeyHash := newMockEncrypter()
	expectedCredential := &store.Credential{
		CredentialID:      rand.Int63(),
		Username:          fmt.Sprintf("testuser%d", rand.Int63()),
		Description:       fmt.Sprintf("description%d", rand.Int63()),
		SSHPrivateKeyHash: testPrivateKeyHash,
	}
	return mockEncrypter, testPrivateKey, expectedCredential
}
