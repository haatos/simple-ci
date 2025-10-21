package settings

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var Settings *AppSettings

func NewSettings() *AppSettings {
	settings := AppSettings{
		SessionExpires: time.Duration(30 * 24 * time.Hour),
		Domain:         getEnvOrDefault("SIMPLECI_DOMAIN", "localhost"),
		Port:           getEnvOrDefault("SIMPLECI_PORT", ":8080"),
		SQLiteDatabase: getEnvOrDefault("SIMPLECI_DB_PATH", "file:.///db.sqlite"),
	}
	if !strings.HasPrefix(settings.Port, ":") {
		settings.Port = ":" + settings.Port
	}
	return &settings
}

func getEnvOrDefault(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return value
}

type AppSettings struct {
	Title                 string
	SQLiteDatabase        string
	SQLiteSessionDatabase string
	Domain                string
	Port                  string
	SessionExpires        time.Duration
	MaxQueuedRuns         int64
}

func (as *AppSettings) BaseURL() string {
	if as.Domain == "localhost" {
		return fmt.Sprintf("http://%s%s", as.Domain, as.Port)
	} else {
		return fmt.Sprintf("https://%s", as.Domain)
	}
}

func (as *AppSettings) SQLiteDbString(readonly bool) string {
	params := make(url.Values)
	params.Add("_journal_mode", "WAL")
	params.Add("_busy_timeout", "5000")
	params.Add("_synchronous", "NORMAL")
	params.Add("_cache_size", "-20000")
	params.Add("_foreign_keys", "ON")
	if readonly {
		params.Add("mode", "ro")
	} else {
		params.Add("_txlock", "IMMEDIATE")
		params.Add("mode", "rwc")
	}

	return as.SQLiteDatabase + "?" + params.Encode()
}

func ReadDotenv(path string) {
	re := regexp.MustCompile(`^[^0-9][A-Z0-9_]+=.+$`)
	f, err := os.Open(path)
	if err != nil {
		log.Fatal("err opening dotenv: ", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) > 0 && line[0] != '#' && re.Match(line) {
			split := strings.Split(string(line), "=")
			name := strings.TrimSpace(split[0])
			value := strings.TrimSpace(split[1])
			value = strings.Trim(value, `"`)
			os.Setenv(name, value)
		}
	}
}
