package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadAndSave(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected LoadConfig to succeed, got %v", err)
	}

	if cfg.Model != DefaultModel {
		t.Errorf("expected default model %s, got %s", DefaultModel, cfg.Model)
	}

	if !cfg.DangerouslySkipPermissions {
		t.Errorf("expected DangerouslySkipPermissions to be true")
	}

	if cfg.IsTelegramConfigured() {
		t.Errorf("expected Telegram not to be configured initially")
	}

	// Test saving
	cfg.TelegramToken = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
	cfg.TelegramChatID = 987654321
	cfg.Model = "gemini-3.7-flash-high"
	cfg.CodingPath = tempHome

	if err := cfg.SaveConfig(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file was written
	cfgFile := filepath.Join(tempHome, ConfigDir, ConfigFile)
	if _, err := os.Stat(cfgFile); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}

	// Reload config
	reloaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	if reloaded.TelegramToken != cfg.TelegramToken {
		t.Errorf("expected token %s, got %s", cfg.TelegramToken, reloaded.TelegramToken)
	}
	if reloaded.TelegramChatID != cfg.TelegramChatID {
		t.Errorf("expected chatID %d, got %d", cfg.TelegramChatID, reloaded.TelegramChatID)
	}
	if reloaded.Model != cfg.Model {
		t.Errorf("expected model %s, got %s", cfg.Model, reloaded.Model)
	}
	if !reloaded.IsTelegramConfigured() {
		t.Errorf("expected Telegram to be configured")
	}
}

func TestConfigEnvOverrides(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	t.Setenv("TELEGRAM_BOT_TOKEN", "test_env_token")
	t.Setenv("TELEGRAM_CHAT_ID", "11223344")
	t.Setenv("MIQA_MODEL", "claude-sonnet-4-6")
	t.Setenv("MIQA_CODING_PATH", tempHome)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.TelegramToken != "test_env_token" {
		t.Errorf("expected env token 'test_env_token', got '%s'", cfg.TelegramToken)
	}
	if cfg.TelegramChatID != 11223344 {
		t.Errorf("expected env chat id 11223344, got %d", cfg.TelegramChatID)
	}
	if cfg.Model != "claude-sonnet-4-6" {
		t.Errorf("expected env model 'claude-sonnet-4-6', got '%s'", cfg.Model)
	}
}
