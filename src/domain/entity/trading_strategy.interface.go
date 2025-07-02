package entity

import (
	"crypgo-machine/src/domain/vo"
)

type TradingDecision string

const (
	Hold TradingDecision = "HOLD"
	Buy  TradingDecision = "BUY"
	Sell TradingDecision = "SELL"
)

type TradingStrategy interface {
	GetName() string
	GetParams() map[string]interface{}
	Decide(klines []vo.Kline, tradingBot *TradingBot) *StrategyAnalysisResult
}
