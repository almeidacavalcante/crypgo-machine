package repository

import (
	"crypgo-machine/src/domain/entity"
	"errors"
	"sync"
)

type TradeBotRepositoryInMemory struct {
	mu   sync.RWMutex
	data map[string]*entity.TradingBot
}

func NewTradeBotRepositoryInMemory() *TradeBotRepositoryInMemory {
	return &TradeBotRepositoryInMemory{
		data: make(map[string]*entity.TradingBot),
	}
}

// Save insere ou atualiza o bot
func (r *TradeBotRepositoryInMemory) Save(bot *entity.TradingBot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[string(bot.Id.GetValue())] = bot
	return nil
}

func (r *TradeBotRepositoryInMemory) Exists(id string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.data[id]
	return exists, nil
}

func (r *TradeBotRepositoryInMemory) GetTradeByID(id string) (*entity.TradingBot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	bot, exists := r.data[id]
	if !exists {
		return nil, nil
	}
	return bot, nil
}

func (r *TradeBotRepositoryInMemory) GetAllTradingBots() ([]*entity.TradingBot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var bots []*entity.TradingBot
	for _, bot := range r.data {
		bots = append(bots, bot)
	}
	return bots, nil
}

func (r *TradeBotRepositoryInMemory) Update(bot *entity.TradingBot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.data[string(bot.Id.GetValue())]; !exists {
		return errors.New("trading bot not found")
	}
	r.data[string(bot.Id.GetValue())] = bot
	return nil
}

func (r *TradeBotRepositoryInMemory) GetTradingBotsByStatus(status entity.Status) ([]*entity.TradingBot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var bots []*entity.TradingBot
	for _, bot := range r.data {
		if bot.GetStatus() == status {
			bots = append(bots, bot)
		}
	}
	return bots, nil
}
