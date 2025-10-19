package internal

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_UnmarshalJSON(t *testing.T) {
	t.Run("success - unmarshal json works as expected", func(t *testing.T) {
		// arrange
		jsonInput := []byte(`{"session_expires_hours": 24, "queue_size": 4}`)
		var config Configuration

		// act
		err := json.Unmarshal(jsonInput, &config)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, time.Duration(24*time.Hour), time.Duration(config.SessionExpiresHours))
	})
}

func TestConfig_MarshalJSON(t *testing.T) {
	t.Run("success - marshal json works as expected", func(t *testing.T) {
		// arrange
		config := Configuration{
			SessionExpiresHours: NewHoursDuration(24),
			QueueSize:           5,
		}

		// act
		b, err := json.Marshal(config)

		// assert
		assert.NoError(t, err)
		assert.Contains(t, string(b), `"session_expires_hours":24`)
		assert.Contains(t, string(b), `"queue_size":5`)
	})
}
