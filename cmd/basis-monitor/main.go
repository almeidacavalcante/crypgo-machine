// Command basis-monitor checks the Deribit futures curve once and sends a
// Telegram alert when the annualized basis reaches a threshold.
//
// It is meant to be run on a schedule (see .github/workflows/basis-monitor.yml).
// It is read-only: it uses public market data and never places orders.
//
// Configuration comes from environment variables only; it does not read .env.
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"crypgo-machine/src/application/basismonitor"
	"crypgo-machine/src/infra/alerting"
	"crypgo-machine/src/infra/deribit"
)

const (
	defaultCurrency       = "BTC"
	defaultThresholdPct   = 8.0
	defaultMinDays        = 30.0
	defaultCooldownHours  = 24.0
	defaultRealertStepPct = 1.0
)

// settings is everything read from the environment.
type settings struct {
	monitor          basismonitor.Config
	stateFile        string
	dryRun           bool
	telegramBotToken string
	telegramChatID   string
}

func main() {
	if err := run(os.Getenv, os.Stdout, time.Now().UTC()); err != nil {
		fmt.Fprintf(os.Stderr, "basis-monitor: %v\n", err)
		os.Exit(1)
	}
}

func run(getenv func(string) string, stdout io.Writer, now time.Time) error {
	cfg, err := loadSettings(getenv)
	if err != nil {
		return err
	}

	var notifier basismonitor.AlertNotifier
	if cfg.dryRun {
		notifier = alerting.NewWriterNotifier(stdout)
	} else {
		sender, err := alerting.NewTelegramSender(cfg.telegramBotToken, cfg.telegramChatID)
		if err != nil {
			return fmt.Errorf("%w (set TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID, or BASIS_DRY_RUN=true)", err)
		}
		notifier = sender
	}

	// Dry runs and runs without a state file must not touch the stored state.
	var store basismonitor.AlertStateStore = alerting.NewMemoryAlertStateStore()
	if cfg.stateFile != "" && !cfg.dryRun {
		store = alerting.NewFileAlertStateStore(cfg.stateFile)
	}

	useCase, err := basismonitor.NewMonitorFuturesBasisUseCase(deribit.NewClient(), notifier, store, cfg.monitor)
	if err != nil {
		return err
	}

	result, runErr := useCase.Execute(now)
	if len(result.Curve) > 0 {
		printReport(stdout, cfg, result, now)
	}
	if outputFile := getenv("GITHUB_OUTPUT"); outputFile != "" && result.Outcome != "" {
		if err := appendLine(outputFile, fmt.Sprintf("outcome=%s", result.Outcome)); err != nil {
			return fmt.Errorf("failed to write the job output: %w", err)
		}
	}
	return runErr
}

func loadSettings(getenv func(string) string) (settings, error) {
	threshold, err := floatSetting(getenv, "BASIS_THRESHOLD_PCT", defaultThresholdPct)
	if err != nil {
		return settings{}, err
	}
	minDays, err := floatSetting(getenv, "BASIS_MIN_DAYS", defaultMinDays)
	if err != nil {
		return settings{}, err
	}
	cooldownHours, err := floatSetting(getenv, "BASIS_COOLDOWN_HOURS", defaultCooldownHours)
	if err != nil {
		return settings{}, err
	}
	realertStep, err := floatSetting(getenv, "BASIS_REALERT_STEP_PCT", defaultRealertStepPct)
	if err != nil {
		return settings{}, err
	}
	dryRun, err := boolSetting(getenv, "BASIS_DRY_RUN")
	if err != nil {
		return settings{}, err
	}

	cfg := settings{
		monitor: basismonitor.Config{
			Venue:           deribit.VenueName,
			Currency:        strings.ToUpper(stringSetting(getenv, "BASIS_CURRENCY", defaultCurrency)),
			ThresholdPct:    threshold,
			MinDaysToExpiry: minDays,
			Cooldown:        time.Duration(cooldownHours * float64(time.Hour)),
			RealertStepPct:  realertStep,
			Locale:          stringSetting(getenv, "BASIS_ALERT_LOCALE", basismonitor.DefaultLocale),
		},
		stateFile:        strings.TrimSpace(getenv("BASIS_STATE_FILE")),
		dryRun:           dryRun,
		telegramBotToken: getenv("TELEGRAM_BOT_TOKEN"),
		telegramChatID:   getenv("TELEGRAM_CHAT_ID"),
	}
	if err := cfg.monitor.Validate(); err != nil {
		return settings{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}

func stringSetting(getenv func(string) string, name, fallback string) string {
	if value := strings.TrimSpace(getenv(name)); value != "" {
		return value
	}
	return fallback
}

func floatSetting(getenv func(string) string, name string, fallback float64) (float64, error) {
	raw := strings.TrimSpace(getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number, got %q", name, raw)
	}
	return value, nil
}

func boolSetting(getenv func(string) string, name string) (bool, error) {
	raw := strings.TrimSpace(getenv(name))
	if raw == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false, got %q", name, raw)
	}
	return value, nil
}

func printReport(out io.Writer, cfg settings, result basismonitor.Result, now time.Time) {
	fmt.Fprintf(out, "%s %s futures basis at %s UTC\n", cfg.monitor.Venue, cfg.monitor.Currency, now.Format("2006-01-02 15:04"))
	fmt.Fprintf(out, "index %.2f USD | alert threshold %.2f%% annualized | min %.0f days to expiry\n\n",
		result.IndexPrice, cfg.monitor.ThresholdPct, cfg.monitor.MinDaysToExpiry)
	fmt.Fprintf(out, "%-14s %6s %11s %9s %11s %9s\n", "contract", "days", "bid", "premium", "annualized", "net")
	for _, basis := range result.Curve {
		fmt.Fprintf(out, "%-14s %6.1f %11.2f %8.3f%% %10.2f%% %8.2f%%\n",
			basis.GetInstrument(), basis.GetDaysToExpiry(), basis.GetFuturesPrice(),
			basis.GetPremium(), basis.GetAnnualized(), basis.GetNetAnnualized())
	}
	fmt.Fprintf(out, "\noutcome: %s (%d contract(s) at or above the threshold)\n", result.Outcome, len(result.Qualifying))
}

func appendLine(path, line string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintln(file, line)
	return err
}
