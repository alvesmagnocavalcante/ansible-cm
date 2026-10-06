package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func clearConfig(t *testing.T) {
	t.Helper()
	for _, key := range []string{"TELEGRAM_BOT_TOKEN", "TELEGRAM_CHAT_ID", "IFC8_TELEGRAM_TOKEN", "IFC8_TELEGRAM_CHAT_ID", "TELEGRAM_TOPIC", "TELEGRAM_TOPIC_ID", "TELEGRAM_TOPIC_TAIBA_ID", "TELEGRAM_TOPIC_CHARME_ID", "TELEGRAM_TOPIC_MAGNA_ID", "TELEGRAM_TOPIC_ACARAIZINHO_ID", "TELEGRAM_TOPIC_WIND_ID", "MONITOR_INTERVAL", "ALERT_REPEAT_INTERVAL", "IFC_LOG_MAX_AGE"} {
		t.Setenv(key, "")
	}
	t.Setenv("IFC_CONFIG_PATH", filepath.Join(t.TempDir(), "SPH.xml"))
	t.Setenv("IFC_LOG_PATH", filepath.Join(t.TempDir(), "SPH.evt"))
}

func TestConfiguration(t *testing.T) {
	clearConfig(t)
	if _, err := loadConfig(true); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(false); err == nil {
		t.Fatal("missing credentials accepted")
	}
	t.Setenv("IFC8_TELEGRAM_TOKEN", "123:legacy")
	t.Setenv("IFC8_TELEGRAM_CHAT_ID", "-123")
	if _, err := loadConfig(false); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TELEGRAM_BOT_TOKEN", "123:canonical")
	cfg, err := loadConfig(false)
	if err != nil || cfg.token != "123:canonical" {
		t.Fatal("canonical setting ignored")
	}
	for _, key := range []string{"MONITOR_INTERVAL", "ALERT_REPEAT_INTERVAL", "IFC_LOG_MAX_AGE"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "-1s")
			if _, err := loadConfig(true); err == nil {
				t.Fatal("negative duration accepted")
			}
		})
	}
	t.Setenv("MONITOR_INTERVAL", "0s")
	if _, err := loadConfig(true); err == nil {
		t.Fatal("zero interval accepted")
	}
}

func TestTopicConfiguration(t *testing.T) {
	clearConfig(t)
	for _, topic := range []string{"Taiba", "charme", "Magna", "acaraizinho", "wind"} {
		t.Run(topic, func(t *testing.T) {
			key := "TELEGRAM_TOPIC_" + strings.ToUpper(topic) + "_ID"
			if _, err := telegramTopicID(topic); err == nil {
				t.Fatal("missing ID accepted")
			}
			t.Setenv(key, "26")
			if id, err := telegramTopicID(topic); err != nil || id != 26 {
				t.Fatalf("%d %v", id, err)
			}
		})
	}
	if _, err := telegramTopicID("other"); err == nil {
		t.Fatal("unknown topic accepted")
	}
	t.Setenv("TELEGRAM_TOPIC_ID", "29")
	if id, err := telegramTopicID(""); err != nil || id != 29 {
		t.Fatal(id, err)
	}
	t.Setenv("TELEGRAM_TOPIC_TAIBA_ID", "26")
	if _, err := telegramTopicID("Taiba"); err == nil {
		t.Fatal("conflicting IDs accepted")
	}
	for _, value := range []string{"0", "-1", "abc", "9223372036854775808"} {
		t.Setenv("TELEGRAM_TOPIC_ID", value)
		if _, err := telegramTopicID(""); err == nil {
			t.Fatal("invalid topic ID accepted")
		}
	}
}

func TestEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := loadEnvFile(path, false); err != nil {
		t.Fatal(err)
	}
	if err := loadEnvFile(path, true); err == nil {
		t.Fatal("explicit missing file accepted")
	}
	t.Setenv("IFC_TEST_ENV", "session")
	if err := os.WriteFile(path, []byte("IFC_TEST_ENV=file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadEnvFile(path, true); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("IFC_TEST_ENV") != "session" {
		t.Fatal("environment overridden")
	}
	secret := "secret-that-must-not-appear"
	if err := os.WriteFile(path, []byte("invalid!="+secret), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadEnvFile(path, true); err == nil || strings.Contains(err.Error(), secret) {
		t.Fatal("bad syntax or redaction")
	}
}
