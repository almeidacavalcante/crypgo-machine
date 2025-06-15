package entity

import (
	"crypgo-machine/src/domain/vo"
	"time"
)

type TradeBot struct {
	ID        vo.UUID
	symbol    vo.Symbol
	quantity  float64
	strategy  TradeStrategy
	status    Status
	createdAt time.Time
}
