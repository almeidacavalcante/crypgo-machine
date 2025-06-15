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
	Name() string
	Decide(klines []vo.Kline) TradingDecision
}
