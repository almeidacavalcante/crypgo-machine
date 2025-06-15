package entity

import vo "crypgo-machine/src/domain/vo"

type TradeStrategy interface {
	Name() string
	Decide(klines []vo.Kline) TradingDecision
}
