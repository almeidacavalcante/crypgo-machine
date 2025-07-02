package entity

import (
	"crypgo-machine/src/domain/vo"
)

type MovingAverageStrategy struct {
	FastWindow    int
	SlowWindow    int
	MinimumSpread vo.MinimumSpread
}

func NewMovingAverageStrategy(fast, slow int) *MovingAverageStrategy {
	// Default minimum spread of 0.5% to avoid whipsaw signals
	minimumSpread, _ := vo.NewMinimumSpread(0.5)
	
	return &MovingAverageStrategy{
		FastWindow:    fast,
		SlowWindow:    slow,
		MinimumSpread: minimumSpread,
	}
}

func NewMovingAverageStrategyWithSpread(fast, slow int, minimumSpread vo.MinimumSpread) *MovingAverageStrategy {
	return &MovingAverageStrategy{
		FastWindow:    fast,
		SlowWindow:    slow,
		MinimumSpread: minimumSpread,
	}
}

func (s *MovingAverageStrategy) GetName() string {
	return "MovingAverage"
}

func (s *MovingAverageStrategy) GetParams() map[string]interface{} {
	return map[string]interface{}{
		"FastWindow":    s.FastWindow,
		"SlowWindow":    s.SlowWindow,
		"MinimumSpread": s.MinimumSpread.GetValue(),
	}
}

func (s *MovingAverageStrategy) Decide(klines []vo.Kline, tradingBot *TradingBot) *StrategyAnalysisResult {
	if len(klines) < s.SlowWindow {
		return NewStrategyAnalysisResult(Hold, map[string]interface{}{
			"fast":   0.0,
			"slow":   0.0,
			"reason": "insufficient_data",
		})
	}

	fast := s.movingAverage(klines, s.FastWindow)
	slow := s.movingAverage(klines, s.SlowWindow)
	currentPrice := klines[len(klines)-1].Close()

	// Check if spread is sufficient to avoid whipsaw signals
	hasSufficientSpread := s.MinimumSpread.HasSufficientSpread(fast, slow)
	
	analysisData := map[string]interface{}{
		"fast":               fast,
		"slow":               slow,
		"currentPrice":       currentPrice,
		"isPositioned":       tradingBot.GetIsPositioned(),
		"hasSufficientSpread": hasSufficientSpread,
		"minimumSpread":      s.MinimumSpread.GetValue(),
		"actualSpread":       s.calculateSpreadPercentage(fast, slow),
	}

	var decision TradingDecision
	
	// INVERTED LOGIC: Buy low, sell high
	if fast < slow && !tradingBot.GetIsPositioned() && hasSufficientSpread {
		decision = Buy
		analysisData["reason"] = "fast_below_slow_buy_low"
	} else if fast > slow && tradingBot.GetIsPositioned() {
		// Sell when price is high
		decision = Sell
		if hasSufficientSpread {
			analysisData["reason"] = "fast_above_slow_sell_high"
		} else {
			analysisData["reason"] = "fast_above_slow_sell_high_no_spread"
		}
	} else {
		decision = Hold
		if fast < slow && !tradingBot.GetIsPositioned() && !hasSufficientSpread {
			analysisData["reason"] = "fast_below_slow_insufficient_spread_wait"
		} else if fast > slow && !tradingBot.GetIsPositioned() {
			analysisData["reason"] = "fast_above_slow_wait_for_dip"
		} else if fast < slow {
			analysisData["reason"] = "fast_below_slow_already_positioned"
		} else {
			analysisData["reason"] = "fast_equals_slow"
		}
	}

	return NewStrategyAnalysisResult(decision, analysisData)
}

func (s *MovingAverageStrategy) movingAverage(klines []vo.Kline, window int) float64 {
	sum := 0.0
	for i := len(klines) - window; i < len(klines); i++ {
		sum += klines[i].Close()
	}
	return sum / float64(window)
}

func (s *MovingAverageStrategy) calculateSpreadPercentage(fast, slow float64) float64 {
	if slow == 0 {
		return 0
	}
	
	// Calculate percentage difference
	percentageDiff := ((fast - slow) / slow) * 100
	if percentageDiff < 0 {
		percentageDiff = -percentageDiff
	}
	
	return percentageDiff
}
