package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type config struct {
	configPath, logPath, token, chatID, topic string
	threadID                                  int64
	interval, repeat, maxLogAge               time.Duration
}

func setting(name, legacy, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	if legacy != "" {
		if value := strings.TrimSpace(os.Getenv(legacy)); value != "" {
			return value
		}
	}
	return fallback
}

func durationSetting(name string, fallback time.Duration, allowZero bool) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration < 0 || (!allowZero && duration == 0) {
		qualifier := "positiva"
		if allowZero {
			qualifier = "nao negativa"
		}
		return 0, fmt.Errorf("%s deve ser uma duracao %s, como 30s ou 1m", name, qualifier)
	}
	return duration, nil
}

func loadConfig(dryRun bool) (config, error) {
	cfg := config{
		configPath: setting("IFC_CONFIG_PATH", "", `C:\Fidelio\Ifc8.Net\IfcApplication\SPH\Ifc8NetConfigSPH.Xml`),
		logPath:    setting("IFC_LOG_PATH", "", `C:\Fidelio\Ifc8.Net\IfcApplication\SPH\M87POS_SPH_Log.evt`),
		token:      setting("TELEGRAM_BOT_TOKEN", "IFC8_TELEGRAM_TOKEN", ""),
		chatID:     setting("TELEGRAM_CHAT_ID", "IFC8_TELEGRAM_CHAT_ID", ""),
		topic:      setting("TELEGRAM_TOPIC", "", ""),
	}
	if !filepath.IsAbs(cfg.configPath) || !filepath.IsAbs(cfg.logPath) {
		return config{}, fmt.Errorf("IFC_CONFIG_PATH e IFC_LOG_PATH devem ser caminhos absolutos")
	}
	if !dryRun && (cfg.token == "" || cfg.chatID == "") {
		return config{}, fmt.Errorf("configure TELEGRAM_BOT_TOKEN e TELEGRAM_CHAT_ID")
	}
	if !dryRun && strings.ContainsAny(cfg.token, "/?# \t\r\n") {
		return config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN contem caracteres invalidos")
	}
	var err error
	if cfg.interval, err = durationSetting("MONITOR_INTERVAL", 30*time.Second, false); err != nil {
		return config{}, err
	}
	if cfg.repeat, err = durationSetting("ALERT_REPEAT_INTERVAL", 5*time.Minute, true); err != nil {
		return config{}, err
	}
	if cfg.maxLogAge, err = durationSetting("IFC_LOG_MAX_AGE", 0, true); err != nil {
		return config{}, err
	}
	if cfg.threadID, err = telegramTopicID(cfg.topic); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func positiveID(name string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(os.Getenv(name)), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s deve ser um ID inteiro positivo", name)
	}
	return value, nil
}

func telegramTopicID(topic string) (int64, error) {
	var id int64
	if topic != "" {
		switch strings.ToLower(topic) {
		case "taiba", "charme", "magna", "acaraizinho", "wind":
		default:
			return 0, fmt.Errorf("TELEGRAM_TOPIC deve ser Taiba, charme, Magna, acaraizinho ou wind")
		}
		var err error
		id, err = positiveID("TELEGRAM_TOPIC_" + strings.ToUpper(topic) + "_ID")
		if err != nil {
			return 0, err
		}
	}
	if strings.TrimSpace(os.Getenv("TELEGRAM_TOPIC_ID")) != "" {
		direct, err := positiveID("TELEGRAM_TOPIC_ID")
		if err != nil {
			return 0, err
		}
		if id != 0 && direct != id {
			return 0, fmt.Errorf("TELEGRAM_TOPIC_ID conflita com TELEGRAM_TOPIC")
		}
		id = direct
	}
	return id, nil
}

func loadEnvFile(path string, required bool) error {
	if err := godotenv.Load(path); err != nil {
		if !required && errors.Is(err, os.ErrNotExist) {
			return nil
		}
		// Parser errors can contain secrets. Never print the original error.
		return fmt.Errorf("nao foi possivel carregar o .env; confira caminho, acesso e sintaxe")
	}
	return nil
}

func defaultEnvPath() string {
	executable, err := os.Executable()
	if err == nil {
		adjacent := filepath.Join(filepath.Dir(executable), ".env")
		info, err := os.Stat(adjacent)
		if err == nil && !info.IsDir() {
			return adjacent
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return adjacent
		}
	}
	return ".env"
}
