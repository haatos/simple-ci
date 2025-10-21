package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurity_AESEncryption(t *testing.T) {
	t.Run("text is encrypted and decrypted", func(t *testing.T) {
		// arrange
		enc := NewAESEncrypter([]byte(GenerateRandomKey(32)))
		expectedText := "this is some text"

		// act
		encrypted := enc.EncryptAES(expectedText)
		decrypted, err := enc.DecryptAES(encrypted)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, expectedText, string(decrypted))
	})
}
