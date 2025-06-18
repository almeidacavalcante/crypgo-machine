package repository

import (
	"crypgo-machine/src/domain/entity"
	"sync"
)

type TradeBotRepositoryInMemory struct {
	mu   sync.RWMutex
	data map[string]*entity.TradeBot
}

func NewTradeBotRepositoryInMemory() *TradeBotRepositoryInMemory {
	return &TradeBotRepositoryInMemory{
		data: make(map[string]*entity.TradeBot),
	}
}

// Save insere ou atualiza o bot
func (r *TradeBotRepositoryInMemory) Save(bot *entity.TradeBot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[string(bot.ID)] = bot
	return nil
}

func (r *TradeBotRepositoryInMemory) Exists(id string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.data[id]
	return exists, nil
}

func (r *TradeBotRepositoryInMemory) GetTradeByID(id string) (*entity.TradeBot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	bot, exists := r.data[id]
	if !exists {
		return nil, nil
	}
	return bot, nil
}
