// Package basismonitor watches the basis of dated futures over the spot index
// and raises an alert when a cash-and-carry trade becomes worth locking in.
//
// It is read-only: it never places orders and needs no exchange credentials.
// The package depends only on the standard library and the domain layer so the
// scheduled job that runs it stays small and fast to build.
package basismonitor

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"crypgo-machine/src/domain/vo"
)

// FuturesQuote is one dated futures contract as reported by the venue.
type FuturesQuote struct {
	Instrument string
	Expiry     time.Time
	// BidPrice is the best price the future can be sold at. Zero means no bid.
	BidPrice float64
	// FeeRate is the cost of the futures leg as a fraction of notional
	// (opening trade plus delivery at expiry).
	FeeRate float64
}

// FuturesMarketSnapshot is the spot index plus every dated future on it.
type FuturesMarketSnapshot struct {
	IndexPrice float64
	Quotes     []FuturesQuote
}

// FuturesMarketDataSource provides the futures curve for a currency.
type FuturesMarketDataSource interface {
	GetDatedFuturesSnapshot(currency string) (FuturesMarketSnapshot, error)
}

// AlertNotifier delivers an alert message to the user.
type AlertNotifier interface {
	Send(message string) error
}

// AlertState is what the monitor remembers between runs to avoid repeating
// the same alert on every execution.
type AlertState struct {
	LastAlertAt             time.Time `json:"last_alert_at"`
	LastAlertBestAnnualized float64   `json:"last_alert_best_annualized"`
}

// AlertStateStore persists AlertState between runs. Load returns the zero
// value when nothing was stored yet.
type AlertStateStore interface {
	Load() (AlertState, error)
	Save(state AlertState) error
}

// Outcome says what a run decided.
type Outcome string

const (
	OutcomeAlertSent      Outcome = "alert_sent"
	OutcomeAlertFailed    Outcome = "alert_failed"
	OutcomeBelowThreshold Outcome = "below_threshold"
	OutcomeCooldown       Outcome = "cooldown"
)

// Config holds the monitor settings. Percentages are in percent (8 means 8%).
type Config struct {
	// Venue is the display name of the exchange, used in the alert text.
	Venue string
	// Currency is the underlying, for example "BTC".
	Currency string
	// ThresholdPct is the gross annualized basis that triggers an alert.
	ThresholdPct float64
	// MinDaysToExpiry ignores contracts too close to expiry, where the
	// annualized figure is dominated by noise.
	MinDaysToExpiry float64
	// Cooldown is the minimum time between two alerts.
	Cooldown time.Duration
	// RealertStepPct lets an alert through during the cooldown when the best
	// annualized basis rose by at least this many percentage points.
	RealertStepPct float64
	// Locale selects the language of the alert text.
	Locale string
}

// Validate reports the first invalid setting.
func (c Config) Validate() error {
	if c.Venue == "" {
		return errors.New("venue cannot be empty")
	}
	if c.Currency == "" {
		return errors.New("currency cannot be empty")
	}
	if c.MinDaysToExpiry < 0 {
		return errors.New("minimum days to expiry cannot be negative")
	}
	if c.Cooldown < 0 {
		return errors.New("cooldown cannot be negative")
	}
	if c.RealertStepPct <= 0 {
		return errors.New("re-alert step must be greater than zero")
	}
	if !IsSupportedLocale(c.Locale) {
		return fmt.Errorf("unsupported locale %q (supported: %v)", c.Locale, SupportedLocales())
	}
	return nil
}

// Result describes one run of the monitor.
type Result struct {
	IndexPrice float64
	// Curve holds every contract that was considered, ordered by expiry.
	Curve []vo.FuturesBasis
	// Qualifying holds the contracts at or above the threshold.
	Qualifying []vo.FuturesBasis
	Outcome    Outcome
}

// MonitorFuturesBasisUseCase computes the futures curve and alerts when the
// annualized basis reaches the configured threshold.
type MonitorFuturesBasisUseCase struct {
	source   FuturesMarketDataSource
	notifier AlertNotifier
	store    AlertStateStore
	config   Config
}

func NewMonitorFuturesBasisUseCase(
	source FuturesMarketDataSource,
	notifier AlertNotifier,
	store AlertStateStore,
	config Config,
) (*MonitorFuturesBasisUseCase, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid basis monitor config: %w", err)
	}
	return &MonitorFuturesBasisUseCase{
		source:   source,
		notifier: notifier,
		store:    store,
		config:   config,
	}, nil
}

// Execute runs one check at the given time.
func (uc *MonitorFuturesBasisUseCase) Execute(now time.Time) (Result, error) {
	snapshot, err := uc.source.GetDatedFuturesSnapshot(uc.config.Currency)
	if err != nil {
		return Result{}, fmt.Errorf("failed to fetch the futures curve: %w", err)
	}

	curve := uc.buildCurve(snapshot, now)
	if len(curve) == 0 {
		return Result{}, fmt.Errorf(
			"no %s dated future has a bid and at least %.0f days to expiry",
			uc.config.Currency, uc.config.MinDaysToExpiry,
		)
	}

	result := Result{IndexPrice: snapshot.IndexPrice, Curve: curve, Outcome: OutcomeBelowThreshold}

	best := 0.0
	for _, basis := range curve {
		if basis.GetAnnualized() < uc.config.ThresholdPct {
			continue
		}
		if len(result.Qualifying) == 0 || basis.GetAnnualized() > best {
			best = basis.GetAnnualized()
		}
		result.Qualifying = append(result.Qualifying, basis)
	}
	if len(result.Qualifying) == 0 {
		return result, nil
	}

	state, err := uc.store.Load()
	if err != nil {
		return result, fmt.Errorf("failed to load the alert state: %w", err)
	}
	if uc.isSuppressed(state, best, now) {
		result.Outcome = OutcomeCooldown
		return result, nil
	}

	message := formatAlert(uc.config, snapshot.IndexPrice, result.Qualifying, now)
	if err := uc.notifier.Send(message); err != nil {
		result.Outcome = OutcomeAlertFailed
		return result, fmt.Errorf("failed to send the basis alert: %w", err)
	}
	result.Outcome = OutcomeAlertSent

	newState := AlertState{LastAlertAt: now, LastAlertBestAnnualized: best}
	if err := uc.store.Save(newState); err != nil {
		return result, fmt.Errorf("alert sent, but saving the alert state failed: %w", err)
	}
	return result, nil
}

// buildCurve turns the quotes into basis figures, dropping contracts that
// cannot be sold (no bid), already expired, or too close to expiry.
func (uc *MonitorFuturesBasisUseCase) buildCurve(snapshot FuturesMarketSnapshot, now time.Time) []vo.FuturesBasis {
	curve := make([]vo.FuturesBasis, 0, len(snapshot.Quotes))
	for _, quote := range snapshot.Quotes {
		if quote.BidPrice <= 0 {
			continue
		}
		basis, err := vo.NewFuturesBasis(
			quote.Instrument, quote.Expiry, now, snapshot.IndexPrice, quote.BidPrice, quote.FeeRate,
		)
		if err != nil {
			continue
		}
		if basis.GetDaysToExpiry() < uc.config.MinDaysToExpiry {
			continue
		}
		curve = append(curve, basis)
	}
	sort.Slice(curve, func(i, j int) bool {
		return curve[i].GetExpiry().Before(curve[j].GetExpiry())
	})
	return curve
}

// isSuppressed reports whether an alert was sent recently enough that this
// one would be a repeat. A clear rise in the basis overrides the cooldown.
func (uc *MonitorFuturesBasisUseCase) isSuppressed(state AlertState, best float64, now time.Time) bool {
	if state.LastAlertAt.IsZero() {
		return false
	}
	if now.Sub(state.LastAlertAt) >= uc.config.Cooldown {
		return false
	}
	return best < state.LastAlertBestAnnualized+uc.config.RealertStepPct
}
