package internal

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/haatos/simple-ci/internal/util"
)

var Config *Configuration

type HoursDuration time.Duration

func NewHoursDuration(hours int64) HoursDuration {
	return HoursDuration(time.Duration(hours) * time.Hour)
}

func (hd HoursDuration) MarshalJSON() ([]byte, error) {
	hours := float64(time.Duration(hd)) / float64(time.Hour)
	return json.Marshal(hours)
}

func (hd *HoursDuration) UnmarshalJSON(data []byte) error {
	var hours float64
	if err := json.Unmarshal(data, &hours); err != nil {
		return err
	}
	*hd = HoursDuration(hours * float64(time.Hour))
	return nil
}

type Configuration struct {
	SessionExpiresHours HoursDuration `json:"session_expires_hours"`
	QueueSize           int64         `json:"queue_size"`
}

func InitializeConfiguration() {
	Config = &Configuration{
		SessionExpiresHours: NewHoursDuration(30 * 24),
		QueueSize:           3,
	}

	configFileExists, _ := util.PathExists("config.json")
	if !configFileExists {
		b, err := json.MarshalIndent(Config, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		configFile, err := os.Create("config.json")
		if err != nil {
			log.Fatal(err)
		}
		if _, err := configFile.Write(b); err != nil {
			log.Fatal(err)
		}
	} else {
		configBytes, err := os.ReadFile("config.json")
		if err != nil {
			log.Fatal(err)
		}
		if err := json.Unmarshal(configBytes, &Config); err != nil {
			log.Fatal(err)
		}
	}
}

func UpdateConfiguration(config *Configuration) error {
	b, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}

	configFile, err := os.Create("config.json")
	if err != nil {
		return err
	}

	if _, err := configFile.Write(b); err != nil {
		return err
	}

	Config = config

	return nil
}
