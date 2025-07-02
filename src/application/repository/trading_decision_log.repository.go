package repository

import "crypgo-machine/src/domain/entity"

type TradingDecisionLogRepository interface {
	Save(log *entity.TradingDecisionLog) error
	GetByTradingBotId(tradingBotId string) ([]*entity.TradingDecisionLog, error)
	GetByTradingBotIdWithLimit(tradingBotId string, limit int) ([]*entity.TradingDecisionLog, error)
}