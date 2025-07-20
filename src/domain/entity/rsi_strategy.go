package entity

import (
	"crypgo-machine/src/domain/vo"
	"fmt"
	"math"
	"time"
)

type RSIStrategy struct {
	Period              int
	OversoldThreshold   float64
	OverboughtThreshold float64
	MinimumSpread       vo.MinimumSpread
	StoplossThreshold   float64
}

func NewRSIStrategy(period int) *RSIStrategy {
	minimumSpread, _ := vo.NewMinimumSpread(0.1)
	
	return &RSIStrategy{
		Period:              period,
		OversoldThreshold:   30.0,
		OverboughtThreshold: 70.0,
		MinimumSpread:       minimumSpread,
		StoplossThreshold:   0.0,
	}
}

func NewRSIStrategyWithCustomThresholds(period int, oversold, overbought float64, minimumSpread vo.MinimumSpread) *RSIStrategy {
	return &RSIStrategy{
		Period:              period,
		OversoldThreshold:   oversold,
		OverboughtThreshold: overbought,
		MinimumSpread:       minimumSpread,
		StoplossThreshold:   0.0,
	}
}

func NewRSIStrategyWithStoploss(period int, oversold, overbought float64, minimumSpread vo.MinimumSpread, stoplossThreshold float64) *RSIStrategy {
	return &RSIStrategy{
		Period:              period,
		OversoldThreshold:   oversold,
		OverboughtThreshold: overbought,
		MinimumSpread:       minimumSpread,
		StoplossThreshold:   stoplossThreshold,
	}
}

func (s *RSIStrategy) GetName() string {
	return "RSI"
}

func (s *RSIStrategy) GetParams() map[string]interface{} {
	return map[string]interface{}{
		"Period":              s.Period,
		"OversoldThreshold":   s.OversoldThreshold,
		"OverboughtThreshold": s.OverboughtThreshold,
		"MinimumSpread":       s.MinimumSpread.GetValue(),
		"StoplossThreshold":   s.StoplossThreshold,
	}
}

func (s *RSIStrategy) Decide(klines []vo.Kline, tradingBot *TradingBot) *StrategyAnalysisResult {
	if len(klines) < s.Period+1 {
		return NewStrategyAnalysisResult(Hold, map[string]interface{}{
			"rsi":    0.0,
			"signal": "NEUTRAL",
			"reason": "insufficient_data",
		})
	}

	rsiResult, err := s.calculateRSI(klines, s.Period)
	if err != nil {
		return NewStrategyAnalysisResult(Hold, map[string]interface{}{
			"rsi":    0.0,
			"signal": "NEUTRAL",
			"reason": "calculation_error",
			"error":  err.Error(),
		})
	}

	currentPrice := klines[len(klines)-1].Close()
	entryPrice := tradingBot.GetEntryPrice()
	possibleProfit := s.calculatePossibleProfit(entryPrice, currentPrice)

	analysisData := map[string]interface{}{
		"rsi":                     rsiResult.Value,
		"signal":                  string(rsiResult.Signal),
		"period":                  rsiResult.Period,
		"currentPrice":            currentPrice,
		"isPositioned":            tradingBot.GetIsPositioned(),
		"oversoldThreshold":       s.OversoldThreshold,
		"overboughtThreshold":     s.OverboughtThreshold,
		"entryPrice":              entryPrice,
		"possibleProfit":          possibleProfit,
		"minimumProfitThreshold":  tradingBot.GetMinimumProfitThreshold(),
		"stoplossThreshold":       s.StoplossThreshold,
	}

	var decision TradingDecision

	// Check for stoploss first if positioned and stoploss is enabled
	if tradingBot.GetIsPositioned() && s.StoplossThreshold > 0 && possibleProfit <= -s.StoplossThreshold {
		decision = Sell
		analysisData["reason"] = "stoploss_triggered"
		return NewStrategyAnalysisResult(decision, analysisData)
	}

	// RSI Logic: Buy when oversold (RSI < 30) and not positioned
	// Sell when overbought (RSI > 70) and positioned with sufficient profit
	if rsiResult.IsOversold() && !tradingBot.GetIsPositioned() {
		decision = Buy
		analysisData["reason"] = "rsi_oversold_buy_signal"
	} else if rsiResult.IsOverbought() && tradingBot.GetIsPositioned() {
		if possibleProfit >= tradingBot.GetMinimumProfitThreshold() {
			decision = Sell
			analysisData["reason"] = "rsi_overbought_sell_with_profit"
		} else {
			decision = Hold
			analysisData["reason"] = "rsi_overbought_hold_insufficient_profit"
		}
	} else {
		decision = Hold
		if rsiResult.IsOversold() && tradingBot.GetIsPositioned() {
			analysisData["reason"] = "rsi_oversold_positioned_holding"
		} else if rsiResult.IsOverbought() && !tradingBot.GetIsPositioned() {
			analysisData["reason"] = "rsi_overbought_wait_for_dip"
		} else if rsiResult.IsNeutral() && !tradingBot.GetIsPositioned() {
			analysisData["reason"] = "rsi_neutral_wait_for_signal"
		} else if rsiResult.IsNeutral() && tradingBot.GetIsPositioned() {
			analysisData["reason"] = "rsi_neutral_positioned_holding"
		} else {
			analysisData["reason"] = "rsi_no_clear_signal"
		}
	}

	return NewStrategyAnalysisResult(decision, analysisData)
}

func (s *RSIStrategy) calculatePossibleProfit(entryPrice, currentPrice float64) float64 {
	if entryPrice == 0 {
		return 0.0
	}

	return ((currentPrice - entryPrice) / entryPrice) * 100
}

func (s *RSIStrategy) calculateRSI(klines []vo.Kline, period int) (*vo.RSIResult, error) {
	if period <= 0 {
		return nil, fmt.Errorf("period must be positive, got: %d", period)
	}
	
	if len(klines) < period+1 {
		return nil, fmt.Errorf("insufficient data: need at least %d klines, got %d", period+1, len(klines))
	}

	// Calculate price changes
	priceChanges := make([]float64, len(klines)-1)
	for i := 1; i < len(klines); i++ {
		priceChanges[i-1] = klines[i].Close() - klines[i-1].Close()
	}

	// Calculate initial average gain and loss for the first RSI value
	initialGain, initialLoss := s.calculateInitialGainLoss(priceChanges[:period])
	
	if initialGain == 0 && initialLoss == 0 {
		return nil, fmt.Errorf("no price changes detected in the data")
	}

	// Calculate RSI using the Wilder's smoothing method
	avgGain := initialGain
	avgLoss := initialLoss
	
	// Apply Wilder's smoothing for remaining periods
	for i := period; i < len(priceChanges); i++ {
		change := priceChanges[i]
		gain := 0.0
		loss := 0.0
		
		if change > 0 {
			gain = change
		} else {
			loss = -change
		}
		
		// Wilder's smoothing formula: new_avg = (prev_avg * (period-1) + new_value) / period
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
	}

	// Calculate RSI
	rsiValue := s.calculateRSIValue(avgGain, avgLoss)
	
	// Use the timestamp of the last kline
	lastKline := klines[len(klines)-1]
	timestamp := time.Unix(lastKline.CloseTime()/1000, 0)
	
	return vo.NewRSIResult(rsiValue, period, timestamp)
}

func (s *RSIStrategy) calculateInitialGainLoss(changes []float64) (float64, float64) {
	var totalGain, totalLoss float64
	
	for _, change := range changes {
		if change > 0 {
			totalGain += change
		} else if change < 0 {
			totalLoss += -change
		}
	}
	
	avgGain := totalGain / float64(len(changes))
	avgLoss := totalLoss / float64(len(changes))
	
	return avgGain, avgLoss
}

func (s *RSIStrategy) calculateRSIValue(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		return 100.0
	}
	
	rs := avgGain / avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))
	
	// Ensure RSI is within valid range
	return math.Max(0, math.Min(100, rsi))
}