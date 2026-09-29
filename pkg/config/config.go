package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration parameters for miqa.
type Config struct {
	TelegramToken              string `json:"telegram_token"`
	TelegramChatID             int64  `json:"telegram_chat_id"`
	Model                      string `json:"model"`
	CodingPath                 string `json:"coding_path"`
	DangerouslySkipPermissions bool   `json:"dangerously_skip_permissions"`
	TaskTimeoutMinutes         int    `json:"task_timeout_minutes"`
}

const (
	DefaultModel = "gemini-3.8-flash-high"
	ConfigDir    = ".config/miqa"
	ConfigFile   = "config.json"
)

// GetConfigPath returns the absolute path to the miqa config file.
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ConfigDir, ConfigFile)
}

// LoadConfig loads the configuration from disk, .env, and environment variables.
func LoadConfig() (*Config, error) {
	// Attempt to load .env if present in current directory or parent
	_ = godotenv.Load()

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	cfg := &Config{
		Model:                      DefaultModel,
		CodingPath:                 cwd,
		DangerouslySkipPermissions: true,
		TaskTimeoutMinutes:         30, // Default 30 min had masa pelaksanaan
	}

	configPath := GetConfigPath()
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	// Environment variable overrides
	if envToken := os.Getenv("TELEGRAM_BOT_TOKEN"); envToken != "" {
		cfg.TelegramToken = strings.TrimSpace(envToken)
	} else if envToken := os.Getenv("TELEGRAM_TOKEN"); envToken != "" {
		cfg.TelegramToken = strings.TrimSpace(envToken)
	}

	if envChatID := os.Getenv("TELEGRAM_CHAT_ID"); envChatID != "" {
		if id, err := strconv.ParseInt(strings.TrimSpace(envChatID), 10, 64); err == nil {
			cfg.TelegramChatID = id
		}
	}

	if envModel := os.Getenv("MIQA_MODEL"); envModel != "" {
		cfg.Model = strings.TrimSpace(envModel)
	}

	if envPath := os.Getenv("MIQA_CODING_PATH"); envPath != "" {
		cfg.CodingPath = strings.TrimSpace(envPath)
	}

	if envTimeout := os.Getenv("MIQA_TIMEOUT_MINUTES"); envTimeout != "" {
		if m, err := strconv.Atoi(strings.TrimSpace(envTimeout)); err == nil && m > 0 {
			cfg.TaskTimeoutMinutes = m
		}
	}

	if cfg.TaskTimeoutMinutes <= 0 {
		cfg.TaskTimeoutMinutes = 30
	}

	// Ensure CodingPath is clean and absolute
	if absPath, err := filepath.Abs(cfg.CodingPath); err == nil {
		cfg.CodingPath = absPath
	}

	return cfg, nil
}

// SaveConfig persists the configuration to ~/.config/miqa/config.json.
func (c *Config) SaveConfig() error {
	configPath := GetConfigPath()
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// IsTelegramConfigured returns true if a Telegram bot token is present.
func (c *Config) IsTelegramConfigured() bool {
	return strings.TrimSpace(c.TelegramToken) != ""
}

// ValidatePath cleans, expands ~, and verifies directory exists.
func ValidatePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "\"'`")
	path = strings.TrimSpace(path)

	if path == "" {
		return "", fmt.Errorf("laluan direktori tidak boleh kosong")
	}

	if strings.HasPrefix(path, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			path = strings.Replace(path, "~", home, 1)
		}
	}

	cleanPath, err := filepath.Abs(path)
	if err != nil {
		cleanPath = filepath.Clean(path)
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", fmt.Errorf("laluan direktori '%s' tidak wujud", cleanPath)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("'%s' bukan sebuah direktori", cleanPath)
	}

	return cleanPath, nil
}
