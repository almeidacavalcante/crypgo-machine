package entity

import (
	"crypgo-machine/src/domain/vo"
	"time"
)

type TradeBot struct {
	ID        vo.UUID
	symbol    vo.Symbol
	quantity  float64
	strategy  TradingStrategy
	status    Status
	createdAt time.Time
}

func NewTradeBot(id vo.UUID, symbol vo.Symbol, quantity float64, strategy TradingStrategy) *TradeBot {
	return &TradeBot{
		ID:        id,
		symbol:    symbol,
		quantity:  quantity,
		strategy:  strategy,
		status:    StatusStopped,
		createdAt: time.Now(),
	}
}

func Restore(id vo.UUID, symbol vo.Symbol, quantity float64, strategy TradingStrategy, status Status, createdAt time.Time) *TradeBot {
	return &TradeBot{
		ID:        id,
		symbol:    symbol,
		quantity:  quantity,
		strategy:  strategy,
		status:    status,
		createdAt: createdAt,
	}
}

func (b *TradeBot) Symbol() vo.Symbol {
	return b.symbol
}

func (b *TradeBot) Quantity() float64 {
	return b.quantity
}

func (b *TradeBot) Strategy() TradingStrategy {
	return b.strategy
}

func (b *TradeBot) Status() Status {
	return b.status
}

func (b *TradeBot) CreatedAt() time.Time {
	return b.createdAt
}
