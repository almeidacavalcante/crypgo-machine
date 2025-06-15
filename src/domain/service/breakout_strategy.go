package service

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
)

type BreakoutStrategy struct {
	Lookback int
}

func NewBreakoutStrategy(lookback int) *BreakoutStrategy {
	return &BreakoutStrategy{
		Lookback: lookback,
	}
}

func (s *BreakoutStrategy) Name() string {
	return "Breakout"
}

func (s *BreakoutStrategy) Decide(klines []vo.Kline) entity.TradingDecision {
	if len(klines) <= s.Lookback {
		return entity.Hold
	}

	last := klines[len(klines)-1]

	highestHigh := klines[len(klines)-s.Lookback-1].High()
	lowestLow := klines[len(klines)-s.Lookback-1].Low()
	for i := len(klines) - s.Lookback - 1; i < len(klines)-1; i++ {
		if klines[i].High() > highestHigh {
			highestHigh = klines[i].High()
		}
		if klines[i].Low() < lowestLow {
			lowestLow = klines[i].Low()
		}
	}

	if last.Close() > highestHigh {
		return entity.Buy
	}
	if last.Close() < lowestLow {
		return entity.Sell
	}
	return entity.Hold
}
