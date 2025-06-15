package service

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
)

type MovingAverageStrategy struct {
	FastWindow int
	SlowWindow int
}

func NewMovingAverageStrategy(fast, slow int) *MovingAverageStrategy {
	return &MovingAverageStrategy{
		FastWindow: fast,
		SlowWindow: slow,
	}
}

func (s *MovingAverageStrategy) Name() string {
	return "MovingAverage"
}

func (s *MovingAverageStrategy) Decide(klines []vo.Kline) entity.TradingDecision {
	if len(klines) < s.SlowWindow {
		return entity.Hold
	}

	fast := s.movingAverage(klines, s.FastWindow)
	slow := s.movingAverage(klines, s.SlowWindow)

	if fast > slow {
		return entity.Buy
	}
	if fast < slow {
		return entity.Sell
	}
	return entity.Hold
}

func (s *MovingAverageStrategy) movingAverage(klines []vo.Kline, window int) float64 {
	sum := 0.0
	for i := len(klines) - window; i < len(klines); i++ {
		sum += klines[i].Close()
	}
	return sum / float64(window)
}
