package entity

import (
	"crypgo-machine/src/domain/vo"
	"fmt"
	"time"
)

type TradingBot struct {
	Id             *vo.EntityId
	symbol         vo.Symbol
	quantity       float64
	strategy       TradingStrategy
	strategyConfig *Strategy
	status         Status
	isPositioned   bool
	intervalSeconds int
	createdAt      time.Time
}

type TradingBotDTO struct {
	Id              string      `json:"id"`
	Symbol          string      `json:"symbol"`
	Quantity        float64     `json:"quantity"`
	Strategy        string      `json:"strategy"`
	StrategyParams  interface{} `json:"strategy_params"`
	Status          string      `json:"status"`
	IsPositioned    bool        `json:"is_positioned"`
	IntervalSeconds int         `json:"interval_seconds"`
	CreatedAt       time.Time   `json:"created_at"`
}

func (b *TradingBot) ToDTO() TradingBotDTO {
	return TradingBotDTO{
		Id:              string(b.Id.GetValue()),
		Symbol:          string(b.symbol.GetValue()),
		Quantity:        b.quantity,
		Strategy:        b.strategy.GetName(),
		StrategyParams:  b.strategy.GetParams(),
		Status:          string(b.status),
		IsPositioned:    b.isPositioned,
		IntervalSeconds: b.intervalSeconds,
		CreatedAt:       b.createdAt,
	}
}

func NewTradingBot(symbol vo.Symbol, quantity float64, strategy TradingStrategy, intervalSeconds int) *TradingBot {
	return &TradingBot{
		Id:              vo.NewEntityId(),
		symbol:          symbol,
		quantity:        quantity,
		strategy:        strategy,
		status:          StatusStopped,
		isPositioned:    false,
		intervalSeconds: intervalSeconds,
		createdAt:       time.Now(),
	}
}

func Restore(id *vo.EntityId, symbol vo.Symbol, quantity float64, strategy TradingStrategy, status Status, isPositioned bool, intervalSeconds int, createdAt time.Time) *TradingBot {
	return &TradingBot{
		Id:              id,
		symbol:          symbol,
		quantity:        quantity,
		strategy:        strategy,
		status:          status,
		isPositioned:    isPositioned,
		intervalSeconds: intervalSeconds,
		createdAt:       createdAt,
	}
}

func BuildStrategy(config *Strategy) (TradingStrategy, error) {
	switch config.GetName() {
	case "MovingAverage":
		fast, _ := config.GetParams()["FastWindow"].(float64)
		slow, _ := config.GetParams()["SlowWindow"].(float64)
		return NewMovingAverageStrategy(int(fast), int(slow)), nil

	default:
		return nil, fmt.Errorf("unknown strategy: %s", config.GetName())
	}
}

func (b *TradingBot) Start() error {
	if b.status != StatusStopped {
		return fmt.Errorf("bot is not in stopped status, current status: %s", b.status)
	}
	b.status = StatusRunning
	return nil
}

func (b *TradingBot) GetIntoPosition() error {
	if b.isPositioned == true {
		return fmt.Errorf("bot is already positioned for this symbol")
	}

	b.isPositioned = true
	return nil
}

func (b *TradingBot) GetOutOfPosition() error {
	if b.isPositioned == false {
		return fmt.Errorf("this bot has no open position for this symbol")
	}

	b.isPositioned = false
	return nil
}

func (b *TradingBot) GetSymbol() vo.Symbol {
	return b.symbol
}

func (b *TradingBot) GetQuantity() float64 {
	return b.quantity
}

func (b *TradingBot) GetStrategy() TradingStrategy {
	return b.strategy
}

func (b *TradingBot) GetStrategyConfig() *Strategy {
	return b.strategyConfig
}

func (b *TradingBot) GetStatus() Status {
	return b.status
}

func (b *TradingBot) GetCreatedAt() time.Time {
	return b.createdAt
}

func (b *TradingBot) GetIsPositioned() bool {
	return b.isPositioned
}

func (b *TradingBot) GetIntervalSeconds() int {
	return b.intervalSeconds
}
