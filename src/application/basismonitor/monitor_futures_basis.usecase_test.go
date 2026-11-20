package basismonitor

import (
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeSource struct {
	snapshot FuturesMarketSnapshot
	err      error
	currency string
}

func (f *fakeSource) GetDatedFuturesSnapshot(currency string) (FuturesMarketSnapshot, error) {
	f.currency = currency
	return f.snapshot, f.err
}

type fakeNotifier struct {
	messages []string
	err      error
}

func (f *fakeNotifier) Send(message string) error {
	if f.err != nil {
		return f.err
	}
	f.messages = append(f.messages, message)
	return nil
}

type fakeStore struct {
	state   AlertState
	loadErr error
	saveErr error
	saves   int
}

func (f *fakeStore) Load() (AlertState, error) { return f.state, f.loadErr }

func (f *fakeStore) Save(state AlertState) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.state = state
	f.saves++
	return nil
}

var testNow = time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)

func testConfig() Config {
	return Config{
		Venue:           "Deribit",
		Currency:        "BTC",
		ThresholdPct:    8,
		MinDaysToExpiry: 7,
		Cooldown:        24 * time.Hour,
		RealertStepPct:  1,
		Locale:          "en",
	}
}

// quoteWithAnnualized builds a quote on a 60,000 index whose gross annualized
// basis is the given percentage.
func quoteWithAnnualized(instrument string, days, annualizedPct float64) FuturesQuote {
	premium := annualizedPct / 100 * days / 365
	return FuturesQuote{
		Instrument: instrument,
		Expiry:     testNow.Add(time.Duration(days * 24 * float64(time.Hour))),
		BidPrice:   60000 * (1 + premium),
		FeeRate:    0.0006,
	}
}

func newUseCase(t *testing.T, quotes []FuturesQuote, store *fakeStore, config Config) (*MonitorFuturesBasisUseCase, *fakeSource, *fakeNotifier) {
	t.Helper()
	source := &fakeSource{snapshot: FuturesMarketSnapshot{IndexPrice: 60000, Quotes: quotes}}
	notifier := &fakeNotifier{}
	useCase, err := NewMonitorFuturesBasisUseCase(source, notifier, store, config)
	if err != nil {
		t.Fatalf("failed to build the use case: %v", err)
	}
	return useCase, source, notifier
}

func TestExecute_BelowThresholdSendsNothing(t *testing.T) {
	store := &fakeStore{}
	useCase, source, notifier := newUseCase(t, []FuturesQuote{
		quoteWithAnnualized("BTC-25DEC26", 35, 4.3),
		quoteWithAnnualized("BTC-26MAR27", 126, 5.3),
	}, store, testConfig())

	result, err := useCase.Execute(testNow)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Outcome != OutcomeBelowThreshold {
		t.Errorf("expected outcome %s, got %s", OutcomeBelowThreshold, result.Outcome)
	}
	if len(result.Curve) != 2 || len(result.Qualifying) != 0 {
		t.Errorf("expected 2 contracts and none qualifying, got %d and %d", len(result.Curve), len(result.Qualifying))
	}
	if result.IndexPrice != 60000 {
		t.Errorf("expected index price 60000, got %v", result.IndexPrice)
	}
	if len(notifier.messages) != 0 {
		t.Errorf("expected no alert, got %d", len(notifier.messages))
	}
	if store.saves != 0 {
		t.Errorf("expected the state to stay untouched, got %d saves", store.saves)
	}
	if source.currency != "BTC" {
		t.Errorf("expected the BTC curve to be requested, got %s", source.currency)
	}
}

func TestExecute_AboveThresholdSendsAlertAndRecordsIt(t *testing.T) {
	store := &fakeStore{}
	useCase, _, notifier := newUseCase(t, []FuturesQuote{
		quoteWithAnnualized("BTC-26MAR27", 126, 9.5),
		quoteWithAnnualized("BTC-25DEC26", 35, 4.3),
		quoteWithAnnualized("BTC-29JAN27", 70, 8.0),
	}, store, testConfig())

	result, err := useCase.Execute(testNow)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Outcome != OutcomeAlertSent {
		t.Fatalf("expected outcome %s, got %s", OutcomeAlertSent, result.Outcome)
	}
	if len(result.Curve) != 3 {
		t.Fatalf("expected 3 contracts in the curve, got %d", len(result.Curve))
	}
	if result.Curve[0].GetInstrument() != "BTC-25DEC26" || result.Curve[2].GetInstrument() != "BTC-26MAR27" {
		t.Errorf("expected the curve ordered by expiry, got %s ... %s",
			result.Curve[0].GetInstrument(), result.Curve[2].GetInstrument())
	}
	if len(result.Qualifying) != 2 {
		t.Fatalf("expected 2 qualifying contracts (threshold is inclusive), got %d", len(result.Qualifying))
	}

	if len(notifier.messages) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(notifier.messages))
	}
	message := notifier.messages[0]
	for _, want := range []string{"BTC FUTURES BASIS ABOVE 8.0% p.a.", "Deribit", "60000.00 USD", "BTC-29JAN27", "BTC-26MAR27", "9.50%", "2026-11-20 08:00"} {
		if !strings.Contains(message, want) {
			t.Errorf("expected the alert to contain %q, got:\n%s", want, message)
		}
	}
	if strings.Contains(message, "BTC-25DEC26") {
		t.Errorf("the alert must not list contracts below the threshold, got:\n%s", message)
	}

	if store.saves != 1 {
		t.Fatalf("expected the state to be saved once, got %d", store.saves)
	}
	if !store.state.LastAlertAt.Equal(testNow) {
		t.Errorf("expected last alert at %v, got %v", testNow, store.state.LastAlertAt)
	}
	if diff := store.state.LastAlertBestAnnualized - 9.5; diff > 1e-6 || diff < -1e-6 {
		t.Errorf("expected best annualized 9.5, got %v", store.state.LastAlertBestAnnualized)
	}
}

func TestExecute_CooldownSuppressesRepeatedAlert(t *testing.T) {
	store := &fakeStore{state: AlertState{LastAlertAt: testNow.Add(-2 * time.Hour), LastAlertBestAnnualized: 9.5}}
	useCase, _, notifier := newUseCase(t, []FuturesQuote{
		quoteWithAnnualized("BTC-26MAR27", 126, 10.2),
	}, store, testConfig())

	result, err := useCase.Execute(testNow)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Outcome != OutcomeCooldown {
		t.Errorf("expected outcome %s, got %s", OutcomeCooldown, result.Outcome)
	}
	if len(result.Qualifying) != 1 {
		t.Errorf("expected the qualifying contract to still be reported, got %d", len(result.Qualifying))
	}
	if len(notifier.messages) != 0 {
		t.Errorf("expected no alert during the cooldown, got %d", len(notifier.messages))
	}
	if store.saves != 0 {
		t.Errorf("expected the state to stay untouched, got %d saves", store.saves)
	}
}

func TestExecute_RiseDuringCooldownAlertsAgain(t *testing.T) {
	store := &fakeStore{state: AlertState{LastAlertAt: testNow.Add(-2 * time.Hour), LastAlertBestAnnualized: 9.5}}
	useCase, _, notifier := newUseCase(t, []FuturesQuote{
		quoteWithAnnualized("BTC-26MAR27", 126, 10.6),
	}, store, testConfig())

	result, err := useCase.Execute(testNow)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Outcome != OutcomeAlertSent {
		t.Errorf("expected outcome %s, got %s", OutcomeAlertSent, result.Outcome)
	}
	if len(notifier.messages) != 1 {
		t.Errorf("expected 1 alert, got %d", len(notifier.messages))
	}
	if !store.state.LastAlertAt.Equal(testNow) {
		t.Errorf("expected the state to move to the new alert, got %v", store.state.LastAlertAt)
	}
}

func TestExecute_AlertsAgainAfterCooldown(t *testing.T) {
	store := &fakeStore{state: AlertState{LastAlertAt: testNow.Add(-24 * time.Hour), LastAlertBestAnnualized: 12}}
	useCase, _, notifier := newUseCase(t, []FuturesQuote{
		quoteWithAnnualized("BTC-26MAR27", 126, 8.5),
	}, store, testConfig())

	result, err := useCase.Execute(testNow)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Outcome != OutcomeAlertSent {
		t.Errorf("expected outcome %s, got %s", OutcomeAlertSent, result.Outcome)
	}
	if len(notifier.messages) != 1 {
		t.Errorf("expected 1 alert, got %d", len(notifier.messages))
	}
}

func TestExecute_IgnoresContractsThatCannotBeTraded(t *testing.T) {
	expired := quoteWithAnnualized("BTC-13NOV26", 30, 50)
	expired.Expiry = testNow.Add(-time.Hour)
	noBid := quoteWithAnnualized("BTC-25JUN27", 217, 50)
	noBid.BidPrice = 0

	store := &fakeStore{}
	useCase, _, notifier := newUseCase(t, []FuturesQuote{
		quoteWithAnnualized("BTC-21NOV26", 1, 50), // too close to expiry
		expired,
		noBid,
		quoteWithAnnualized("BTC-26MAR27", 126, 5),
	}, store, testConfig())

	result, err := useCase.Execute(testNow)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Curve) != 1 || result.Curve[0].GetInstrument() != "BTC-26MAR27" {
		t.Fatalf("expected only BTC-26MAR27 in the curve, got %d contract(s)", len(result.Curve))
	}
	if len(notifier.messages) != 0 {
		t.Errorf("expected no alert, got %d", len(notifier.messages))
	}
}

func TestExecute_FailsWhenNoContractIsUsable(t *testing.T) {
	store := &fakeStore{}
	useCase, _, _ := newUseCase(t, []FuturesQuote{
		quoteWithAnnualized("BTC-21NOV26", 1, 5),
	}, store, testConfig())

	if _, err := useCase.Execute(testNow); err == nil {
		t.Fatal("expected an error when no contract can be evaluated")
	}
}

func TestExecute_PropagatesSourceError(t *testing.T) {
	source := &fakeSource{err: errors.New("exchange unavailable")}
	notifier := &fakeNotifier{}
	useCase, err := NewMonitorFuturesBasisUseCase(source, notifier, &fakeStore{}, testConfig())
	if err != nil {
		t.Fatalf("failed to build the use case: %v", err)
	}

	_, err = useCase.Execute(testNow)
	if err == nil || !strings.Contains(err.Error(), "exchange unavailable") {
		t.Fatalf("expected the source error to be returned, got %v", err)
	}
	if len(notifier.messages) != 0 {
		t.Errorf("expected no alert, got %d", len(notifier.messages))
	}
}

func TestExecute_FailedAlertIsNotRecorded(t *testing.T) {
	store := &fakeStore{}
	useCase, _, notifier := newUseCase(t, []FuturesQuote{
		quoteWithAnnualized("BTC-26MAR27", 126, 9.5),
	}, store, testConfig())
	notifier.err = errors.New("telegram is down")

	result, err := useCase.Execute(testNow)
	if err == nil || !strings.Contains(err.Error(), "telegram is down") {
		t.Fatalf("expected the notifier error to be returned, got %v", err)
	}
	if result.Outcome != OutcomeAlertFailed {
		t.Errorf("expected outcome %s, got %s", OutcomeAlertFailed, result.Outcome)
	}
	if store.saves != 0 {
		t.Errorf("a failed alert must not start the cooldown, got %d saves", store.saves)
	}
}

func TestExecute_ReportsStateErrors(t *testing.T) {
	quotes := []FuturesQuote{quoteWithAnnualized("BTC-26MAR27", 126, 9.5)}

	t.Run("load", func(t *testing.T) {
		store := &fakeStore{loadErr: errors.New("corrupt state")}
		useCase, _, notifier := newUseCase(t, quotes, store, testConfig())

		if _, err := useCase.Execute(testNow); err == nil {
			t.Fatal("expected an error when the state cannot be loaded")
		}
		if len(notifier.messages) != 0 {
			t.Errorf("expected no alert, got %d", len(notifier.messages))
		}
	})

	t.Run("save", func(t *testing.T) {
		store := &fakeStore{saveErr: errors.New("disk full")}
		useCase, _, notifier := newUseCase(t, quotes, store, testConfig())

		result, err := useCase.Execute(testNow)
		if err == nil {
			t.Fatal("expected an error when the state cannot be saved")
		}
		if result.Outcome != OutcomeAlertSent || len(notifier.messages) != 1 {
			t.Errorf("the alert was delivered and must be reported as sent, got outcome %s", result.Outcome)
		}
	})
}

func TestFormatAlert_DefaultLocaleIsPortuguese(t *testing.T) {
	config := testConfig()
	config.Locale = DefaultLocale
	store := &fakeStore{}
	useCase, _, notifier := newUseCase(t, []FuturesQuote{
		quoteWithAnnualized("BTC-26MAR27", 126, 9.5),
	}, store, config)

	if _, err := useCase.Execute(testNow); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(notifier.messages) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(notifier.messages))
	}
	for _, want := range []string{"BASE DE FUTUROS BTC ACIMA DE 8.0% a.a.", "126 dias", "Anualizado: <b>9.50%</b>", "20/11/2026 08:00"} {
		if !strings.Contains(notifier.messages[0], want) {
			t.Errorf("expected the alert to contain %q, got:\n%s", want, notifier.messages[0])
		}
	}
}

func TestConfigValidate(t *testing.T) {
	if err := testConfig().Validate(); err != nil {
		t.Fatalf("expected the test config to be valid, got %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{"empty venue", func(c *Config) { c.Venue = "" }},
		{"empty currency", func(c *Config) { c.Currency = "" }},
		{"negative min days", func(c *Config) { c.MinDaysToExpiry = -1 }},
		{"negative cooldown", func(c *Config) { c.Cooldown = -time.Second }},
		{"zero re-alert step", func(c *Config) { c.RealertStepPct = 0 }},
		{"unknown locale", func(c *Config) { c.Locale = "fr" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := testConfig()
			tc.mutate(&config)
			if err := config.Validate(); err == nil {
				t.Error("expected a validation error, got nil")
			}
			if _, err := NewMonitorFuturesBasisUseCase(&fakeSource{}, &fakeNotifier{}, &fakeStore{}, config); err == nil {
				t.Error("expected the constructor to reject the config")
			}
		})
	}
}

func TestSupportedLocales(t *testing.T) {
	locales := SupportedLocales()
	if len(locales) != 2 || locales[0] != "en" || locales[1] != "pt-BR" {
		t.Errorf("expected [en pt-BR], got %v", locales)
	}
	if !IsSupportedLocale(DefaultLocale) {
		t.Errorf("the default locale %s must be supported", DefaultLocale)
	}
}
