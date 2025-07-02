package vo

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Kline struct {
	open      float64
	close     float64
	high      float64
	low       float64
	volume    float64
	closeTime int64
}

func (k Kline) Open() float64    { return k.open }
func (k Kline) Close() float64   { return k.close }
func (k Kline) High() float64    { return k.high }
func (k Kline) Low() float64     { return k.low }
func (k Kline) Volume() float64  { return k.volume }
func (k Kline) CloseTime() int64 { return k.closeTime }

func NewKline(open, close, high, low, volume float64, closeTime int64) (Kline, error) {
	k := Kline{
		open:      open,
		close:     close,
		high:      high,
		low:       low,
		volume:    volume,
		closeTime: closeTime,
	}
	if err := k.validate(); err != nil {
		return Kline{}, err
	}
	return k, nil
}

func (k Kline) validate() error {
	if k.open < 0 {
		return errors.New("open cannot be negative")
	}
	if k.close < 0 {
		return errors.New("close cannot be negative")
	}
	if k.high < 0 {
		return errors.New("high cannot be negative")
	}
	if k.low < 0 {
		return errors.New("low cannot be negative")
	}
	if k.volume < 0 {
		return errors.New("volume cannot be negative")
	}
	if k.closeTime <= 0 {
		return errors.New("closeTime must be positive")
	}
	if k.high < k.open || k.high < k.close || k.high < k.low {
		return fmt.Errorf("high must be ≥ open, close, and low")
	}
	if k.low > k.open || k.low > k.close || k.low > k.high {
		return fmt.Errorf("low must be ≤ open, close, and high")
	}
	if k.high < k.low {
		return fmt.Errorf("high must be ≥ low")
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling for Kline
func (k Kline) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Open      float64 `json:"open"`
		Close     float64 `json:"close"`
		High      float64 `json:"high"`
		Low       float64 `json:"low"`
		Volume    float64 `json:"volume"`
		CloseTime int64   `json:"closeTime"`
	}{
		Open:      k.open,
		Close:     k.close,
		High:      k.high,
		Low:       k.low,
		Volume:    k.volume,
		CloseTime: k.closeTime,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Kline
func (k *Kline) UnmarshalJSON(data []byte) error {
	var aux struct {
		Open      float64 `json:"open"`
		Close     float64 `json:"close"`
		High      float64 `json:"high"`
		Low       float64 `json:"low"`
		Volume    float64 `json:"volume"`
		CloseTime int64   `json:"closeTime"`
	}
	
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	
	kline, err := NewKline(aux.Open, aux.Close, aux.High, aux.Low, aux.Volume, aux.CloseTime)
	if err != nil {
		return err
	}
	
	*k = kline
	return nil
}
