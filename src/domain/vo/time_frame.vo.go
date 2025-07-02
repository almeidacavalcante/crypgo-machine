package vo

import "fmt"

type Timeframe struct {
	value string
}

var allowedTimeframes = map[string]struct{}{
	"1m":  {},
	"5m":  {},
	"10m": {},
	"15m": {},
	"1h":  {},
	"4h":  {},
	"1d":  {},
}

func NewTimeframe(value string) (Timeframe, error) {
	if _, ok := allowedTimeframes[value]; !ok {
		return Timeframe{}, fmt.Errorf("invalid timeframe %s", value)
	}
	return Timeframe{value: value}, nil
}

func (t Timeframe) GetValue() string {
	return t.value
}
