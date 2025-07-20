package entity

import (
	"crypgo-machine/src/domain/vo"
	"fmt"
)

type MovingAverageStrategy struct {
	FastWindow        int
	SlowWindow        int
	MinimumSpread     vo.MinimumSpread
	StoplossThreshold float64
}

func NewMovingAverageStrategy(fast, slow int) *MovingAverageStrategy {

	minimumSpread, _ := vo.NewMinimumSpread(0.1)

	return &MovingAverageStrategy{
		FastWindow:        fast,
		SlowWindow:        slow,
		MinimumSpread:     minimumSpread,
		StoplossThreshold: 0.0,
	}
}

func NewMovingAverageStrategyWithSpread(fast, slow int, minimumSpread vo.MinimumSpread) *MovingAverageStrategy {
	return &MovingAverageStrategy{
		FastWindow:        fast,
		SlowWindow:        slow,
		MinimumSpread:     minimumSpread,
		StoplossThreshold: 0.0,
	}
}

func NewMovingAverageStrategyWithStoploss(fast, slow int, minimumSpread vo.MinimumSpread, stoplossThreshold float64) *MovingAverageStrategy {
	return &MovingAverageStrategy{
		FastWindow:        fast,
		SlowWindow:        slow,
		MinimumSpread:     minimumSpread,
		StoplossThreshold: stoplossThreshold,
	}
}

func (s *MovingAverageStrategy) GetName() string {
	return "MovingAverage"
}

func (s *MovingAverageStrategy) GetParams() map[string]interface{} {
	return map[string]interface{}{
		"FastWindow":        s.FastWindow,
		"SlowWindow":        s.SlowWindow,
		"MinimumSpread":     s.MinimumSpread.GetValue(),
		"StoplossThreshold": s.StoplossThreshold,
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

	hasSufficientSpread := s.MinimumSpread.HasSufficientSpread(fast, slow)

	entryPrice := tradingBot.GetEntryPrice()
	possibleProfit := s.calculatePossibleProfit(entryPrice, currentPrice)

	analysisData := map[string]interface{}{
		"fast":                   fast,
		"slow":                   slow,
		"currentPrice":           currentPrice,
		"isPositioned":           tradingBot.GetIsPositioned(),
		"hasSufficientSpread":    hasSufficientSpread,
		"minimumSpread":          s.MinimumSpread.GetValue(),
		"actualSpread":           s.calculateSpreadPercentage(fast, slow),
		"entryPrice":             entryPrice,
		"possibleProfit":         possibleProfit,
		"minimumProfitThreshold": tradingBot.GetMinimumProfitThreshold(),
		"stoplossThreshold":      s.StoplossThreshold,
	}

	var decision TradingDecision

	// Check for stoploss first if positioned and stoploss is enabled
	if tradingBot.GetIsPositioned() && s.StoplossThreshold > 0 && possibleProfit <= -s.StoplossThreshold {
		fmt.Printf("🚨 STOPLOSS TRIGGERED! Price: %.2f | Entry: %.2f | Loss: %.2f%% | Threshold: %.2f%%\n", 
			currentPrice, entryPrice, possibleProfit, s.StoplossThreshold)
		decision = Sell
		analysisData["reason"] = "stoploss_triggered"
		return NewStrategyAnalysisResult(decision, analysisData)
	}

	if fast < slow && !tradingBot.GetIsPositioned() && hasSufficientSpread {
		decision = Buy
		analysisData["reason"] = "fast_below_slow_buy_low"
	} else if fast > slow && tradingBot.GetIsPositioned() {

		if possibleProfit >= tradingBot.GetMinimumProfitThreshold() {
			decision = Sell
			analysisData["reason"] = "fast_above_slow_sell_high_with_profit"
		} else {
			decision = Hold
			analysisData["reason"] = "fast_above_slow_hold_insufficient_profit"
		}
	} else {
		decision = Hold
		if fast < slow && !tradingBot.GetIsPositioned() && !hasSufficientSpread {
			analysisData["reason"] = "fast_below_slow_insufficient_spread_wait"
		} else if fast > slow && !tradingBot.GetIsPositioned() {
			analysisData["reason"] = "fast_above_slow_wait_for_dip"
		} else if fast < slow && tradingBot.GetIsPositioned() {
			analysisData["reason"] = "fast_below_slow_positioned_holding"
		} else if fast > slow && tradingBot.GetIsPositioned() {
			analysisData["reason"] = "fast_above_slow_positioned_waiting_for_profit"
		} else {
			analysisData["reason"] = "fast_equals_slow_neutral"
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

	percentageDiff := ((fast - slow) / slow) * 100
	if percentageDiff < 0 {
		percentageDiff = -percentageDiff
	}

	return percentageDiff
}

func (s *MovingAverageStrategy) calculatePossibleProfit(entryPrice, currentPrice float64) float64 {
	if entryPrice == 0 {
		return 0.0
	}

	return ((currentPrice - entryPrice) / entryPrice) * 100
}
