package entity

import (
	"crypgo-machine/src/domain/vo"
	"fmt"
	"strings"
)

type TradingDecision string

const (
	Hold TradingDecision = "HOLD"
	Buy  TradingDecision = "BUY"
	Sell TradingDecision = "SELL"
)

// ParseTradingDecision converts a string to TradingDecision
func ParseTradingDecision(s string) (TradingDecision, error) {
	switch strings.ToUpper(s) {
	case "HOLD":
		return Hold, nil
	case "BUY":
		return Buy, nil
	case "SELL":
		return Sell, nil
	default:
		return "", fmt.Errorf("invalid trading decision: %s", s)
	}
}

type TradingStrategy interface {
	GetName() string
	GetParams() map[string]interface{}
	Decide(klines []vo.Kline, tradingBot *TradingBot) *StrategyAnalysisResult
}
