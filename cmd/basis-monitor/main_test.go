package main

import (
	"strings"
	"testing"
	"time"
)

func envFrom(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

func TestLoadSettings_Defaults(t *testing.T) {
	cfg, err := loadSettings(envFrom(nil))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.monitor.Venue != "Deribit" {
		t.Errorf("expected venue Deribit, got %s", cfg.monitor.Venue)
	}
	if cfg.monitor.Currency != "BTC" {
		t.Errorf("expected currency BTC, got %s", cfg.monitor.Currency)
	}
	if cfg.monitor.ThresholdPct != 8 {
		t.Errorf("expected threshold 8, got %v", cfg.monitor.ThresholdPct)
	}
	if cfg.monitor.MinDaysToExpiry != 30 {
		t.Errorf("expected min days 30, got %v", cfg.monitor.MinDaysToExpiry)
	}
	if cfg.monitor.Cooldown != 24*time.Hour {
		t.Errorf("expected cooldown 24h, got %v", cfg.monitor.Cooldown)
	}
	if cfg.monitor.RealertStepPct != 1 {
		t.Errorf("expected re-alert step 1, got %v", cfg.monitor.RealertStepPct)
	}
	if cfg.monitor.Locale != "pt-BR" {
		t.Errorf("expected locale pt-BR, got %s", cfg.monitor.Locale)
	}
	if cfg.dryRun {
		t.Error("dry run must default to false")
	}
	if cfg.stateFile != "" {
		t.Errorf("expected no state file, got %s", cfg.stateFile)
	}
}

func TestLoadSettings_ReadsOverrides(t *testing.T) {
	cfg, err := loadSettings(envFrom(map[string]string{
		"BASIS_CURRENCY":         " eth ",
		"BASIS_THRESHOLD_PCT":    "6.5",
		"BASIS_MIN_DAYS":         "14",
		"BASIS_COOLDOWN_HOURS":   "0.5",
		"BASIS_REALERT_STEP_PCT": "2",
		"BASIS_ALERT_LOCALE":     "en",
		"BASIS_STATE_FILE":       " state/alert.json ",
		"BASIS_DRY_RUN":          "true",
		"TELEGRAM_BOT_TOKEN":     "token",
		"TELEGRAM_CHAT_ID":       "42",
	}))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.monitor.Currency != "ETH" {
		t.Errorf("expected currency ETH, got %s", cfg.monitor.Currency)
	}
	if cfg.monitor.ThresholdPct != 6.5 {
		t.Errorf("expected threshold 6.5, got %v", cfg.monitor.ThresholdPct)
	}
	if cfg.monitor.MinDaysToExpiry != 14 {
		t.Errorf("expected min days 14, got %v", cfg.monitor.MinDaysToExpiry)
	}
	if cfg.monitor.Cooldown != 30*time.Minute {
		t.Errorf("expected cooldown 30m, got %v", cfg.monitor.Cooldown)
	}
	if cfg.monitor.RealertStepPct != 2 {
		t.Errorf("expected re-alert step 2, got %v", cfg.monitor.RealertStepPct)
	}
	if cfg.monitor.Locale != "en" {
		t.Errorf("expected locale en, got %s", cfg.monitor.Locale)
	}
	if cfg.stateFile != "state/alert.json" {
		t.Errorf("expected trimmed state file, got %q", cfg.stateFile)
	}
	if !cfg.dryRun {
		t.Error("expected dry run to be enabled")
	}
	if cfg.telegramBotToken != "token" || cfg.telegramChatID != "42" {
		t.Error("expected telegram settings to be read")
	}
}

func TestLoadSettings_RejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name  string
		env   map[string]string
		wants string
	}{
		{"threshold is not a number", map[string]string{"BASIS_THRESHOLD_PCT": "eight"}, "BASIS_THRESHOLD_PCT"},
		{"min days is not a number", map[string]string{"BASIS_MIN_DAYS": "x"}, "BASIS_MIN_DAYS"},
		{"cooldown is not a number", map[string]string{"BASIS_COOLDOWN_HOURS": "1h"}, "BASIS_COOLDOWN_HOURS"},
		{"re-alert step is not a number", map[string]string{"BASIS_REALERT_STEP_PCT": "-"}, "BASIS_REALERT_STEP_PCT"},
		{"dry run is not a boolean", map[string]string{"BASIS_DRY_RUN": "maybe"}, "BASIS_DRY_RUN"},
		{"unsupported locale", map[string]string{"BASIS_ALERT_LOCALE": "fr"}, "unsupported locale"},
		{"negative min days", map[string]string{"BASIS_MIN_DAYS": "-1"}, "minimum days"},
		{"zero re-alert step", map[string]string{"BASIS_REALERT_STEP_PCT": "0"}, "re-alert step"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadSettings(envFrom(tc.env))
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wants) {
				t.Errorf("expected error to mention %q, got %v", tc.wants, err)
			}
		})
	}
}

func TestRun_RequiresTelegramSettingsUnlessDryRun(t *testing.T) {
	var out strings.Builder
	err := run(envFrom(nil), &out, time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected an error when Telegram is not configured")
	}
	if !strings.Contains(err.Error(), "TELEGRAM_BOT_TOKEN") {
		t.Errorf("expected error to name the missing settings, got %v", err)
	}
}
