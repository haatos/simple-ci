package settings

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSettings_ReadDotenv(t *testing.T) {
	t.Run("success - .env files is read into env variables", func(t *testing.T) {
		// arrange
		testDotEnvFile := ".env.test"
		f, err := os.Create(testDotEnvFile)
		if err != nil {
			t.Error(err)
		}
		lines := []string{
			`#COMMENTED=asdf`,
			`SIMPLE_CI_TEST=1234`,
			``,
			`SIMPLE_CI_TEST2= 2345 `,
		}
		for _, line := range lines {
			f.Write([]byte(line + "\n"))
		}
		f.Close()
		defer os.Remove(testDotEnvFile)

		// act
		ReadDotenv(testDotEnvFile)

		// assert
		assert.Equal(t, os.Getenv("SIMPLE_CI_TEST"), "1234")
		assert.Equal(t, os.Getenv("SIMPLE_CI_TEST2"), "2345")
	})
}
