package vo

import (
	"errors"
	"time"
)

const (
	daysPerYear  = 365.0
	hoursPerDay  = 24.0
	percentScale = 100.0
)

// FuturesBasis is the premium of a dated futures contract over the spot
// index, as seen by someone opening a cash-and-carry trade (long spot, short
// the future). All percentages are expressed as percent, so 5.25 means 5.25%.
type FuturesBasis struct {
	instrument    string
	expiry        time.Time
	indexPrice    float64
	futuresPrice  float64
	daysToExpiry  float64
	premium       float64
	annualized    float64
	netPremium    float64
	netAnnualized float64
}

// NewFuturesBasis computes the basis of a futures contract at a point in time.
//
// futuresPrice is the price the future can be sold at (the best bid).
// feeRate is the total cost of the futures leg as a fraction of notional
// (for example 0.0006 for 0.06%), used only for the net figures.
//
// The annualized figure uses simple (not compounded) annualization, which is
// the market convention for quoting futures basis.
func NewFuturesBasis(instrument string, expiry, now time.Time, indexPrice, futuresPrice, feeRate float64) (FuturesBasis, error) {
	if instrument == "" {
		return FuturesBasis{}, errors.New("instrument cannot be empty")
	}
	if indexPrice <= 0 {
		return FuturesBasis{}, errors.New("index price must be greater than zero")
	}
	if futuresPrice <= 0 {
		return FuturesBasis{}, errors.New("futures price must be greater than zero")
	}
	if feeRate < 0 {
		return FuturesBasis{}, errors.New("fee rate cannot be negative")
	}
	if !expiry.After(now) {
		return FuturesBasis{}, errors.New("expiry must be in the future")
	}

	daysToExpiry := expiry.Sub(now).Hours() / hoursPerDay
	premium := (futuresPrice/indexPrice - 1) * percentScale
	netPremium := premium - feeRate*percentScale

	return FuturesBasis{
		instrument:    instrument,
		expiry:        expiry,
		indexPrice:    indexPrice,
		futuresPrice:  futuresPrice,
		daysToExpiry:  daysToExpiry,
		premium:       premium,
		annualized:    premium * daysPerYear / daysToExpiry,
		netPremium:    netPremium,
		netAnnualized: netPremium * daysPerYear / daysToExpiry,
	}, nil
}

func (b FuturesBasis) GetInstrument() string { return b.instrument }

func (b FuturesBasis) GetExpiry() time.Time { return b.expiry }

func (b FuturesBasis) GetIndexPrice() float64 { return b.indexPrice }

func (b FuturesBasis) GetFuturesPrice() float64 { return b.futuresPrice }

func (b FuturesBasis) GetDaysToExpiry() float64 { return b.daysToExpiry }

// GetPremium returns the premium over the index for the whole period, in percent.
func (b FuturesBasis) GetPremium() float64 { return b.premium }

// GetAnnualized returns the gross annualized basis, in percent.
func (b FuturesBasis) GetAnnualized() float64 { return b.annualized }

// GetNetPremium returns the premium after the futures leg fees, in percent.
func (b FuturesBasis) GetNetPremium() float64 { return b.netPremium }

// GetNetAnnualized returns the annualized basis after the futures leg fees, in percent.
func (b FuturesBasis) GetNetAnnualized() float64 { return b.netAnnualized }
