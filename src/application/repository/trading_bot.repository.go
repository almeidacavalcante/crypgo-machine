package repository

import "crypgo-machine/src/domain/entity"

type TradingBotRepository interface {
	Save(trade *entity.TradingBot) error
	Update(trade *entity.TradingBot) error
	GetTradeByID(id string) (*entity.TradingBot, error)
	Exists(id string) (bool, error)
	GetAllTradingBots() ([]*entity.TradingBot, error)
}
