package repository

import "crypgo-machine/src/domain/entity"

type TradeBotRepository interface {
	Save(trade *entity.TradeBot) error
	GetTradeByID(id string) (*entity.TradeBot, error)
	Exists(id string) (bool, error)
}
