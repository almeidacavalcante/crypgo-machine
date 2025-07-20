package entity

import (
	"crypgo-machine/src/domain/vo"
	"fmt"
	"time"
)

type TradingBot struct {
	Id                     *vo.EntityId
	symbol                 vo.Symbol
	quantity               float64
	strategy               TradingStrategy
	strategyConfig         *Strategy
	status                 Status
	isPositioned           bool
	entryPrice             float64 // Price when position was opened
	actualQuantityHeld     float64 // Actual quantity held after fees (for sell orders)
	intervalSeconds        int
	initialCapital         float64
	tradeAmount            float64
	currency               string
	tradingFees            float64
	minimumProfitThreshold float64
	useFixedQuantity       bool    // true = use quantity field, false = use tradeAmount to calculate dynamic quantity
	createdAt              time.Time
}

type TradingBotDTO struct {
	Id                     string      `json:"id"`
	Symbol                 string      `json:"symbol"`
	Quantity               float64     `json:"quantity"`
	Strategy               string      `json:"strategy"`
	StrategyParams         interface{} `json:"strategy_params"`
	Status                 string      `json:"status"`
	IsPositioned           bool        `json:"is_positioned"`
	EntryPrice             *float64    `json:"entry_price"`
	ActualQuantityHeld     float64     `json:"actual_quantity_held"`
	IntervalSeconds        int         `json:"interval_seconds"`
	InitialCapital         float64     `json:"initial_capital"`
	TradeAmount            float64     `json:"trade_amount"`
	Currency               string      `json:"currency"`
	TradingFees            float64     `json:"trading_fees"`
	MinimumProfitThreshold float64     `json:"minimum_profit_threshold"`
	UseFixedQuantity       bool        `json:"use_fixed_quantity"`
	CreatedAt              time.Time   `json:"created_at"`
}

func (b *TradingBot) ToDTO() TradingBotDTO {
	var entryPrice *float64
	if b.entryPrice > 0 {
		entryPrice = &b.entryPrice
	}
	
	return TradingBotDTO{
		Id:                     string(b.Id.GetValue()),
		Symbol:                 string(b.symbol.GetValue()),
		Quantity:               b.quantity,
		Strategy:               b.strategy.GetName(),
		StrategyParams:         b.strategy.GetParams(),
		Status:                 string(b.status),
		IsPositioned:           b.isPositioned,
		EntryPrice:             entryPrice,
		ActualQuantityHeld:     b.actualQuantityHeld,
		IntervalSeconds:        b.intervalSeconds,
		InitialCapital:         b.initialCapital,
		TradeAmount:            b.tradeAmount,
		Currency:               b.currency,
		TradingFees:            b.tradingFees,
		MinimumProfitThreshold: b.minimumProfitThreshold,
		UseFixedQuantity:       b.useFixedQuantity,
		CreatedAt:              b.createdAt,
	}
}

func NewTradingBot(symbol vo.Symbol, quantity float64, strategy TradingStrategy, intervalSeconds int, initialCapital float64, tradeAmount float64, currency string, tradingFees float64, minimumProfitThreshold float64, useFixedQuantity bool) *TradingBot {
	return &TradingBot{
		Id:                     vo.NewEntityId(),
		symbol:                 symbol,
		quantity:               quantity,
		strategy:               strategy,
		status:                 StatusStopped,
		isPositioned:           false,
		intervalSeconds:        intervalSeconds,
		initialCapital:         initialCapital,
		tradeAmount:            tradeAmount,
		currency:               currency,
		tradingFees:            tradingFees,
		minimumProfitThreshold: minimumProfitThreshold,
		useFixedQuantity:       useFixedQuantity,
		createdAt:              time.Now(),
	}
}

func Restore(id *vo.EntityId, symbol vo.Symbol, quantity float64, strategy TradingStrategy, status Status, isPositioned bool, intervalSeconds int, initialCapital float64, tradeAmount float64, currency string, tradingFees float64, minimumProfitThreshold float64, entryPrice float64, actualQuantityHeld float64, useFixedQuantity bool, createdAt time.Time) *TradingBot {
	return &TradingBot{
		Id:                     id,
		symbol:                 symbol,
		quantity:               quantity,
		strategy:               strategy,
		status:                 status,
		isPositioned:           isPositioned,
		intervalSeconds:        intervalSeconds,
		initialCapital:         initialCapital,
		tradeAmount:            tradeAmount,
		currency:               currency,
		tradingFees:            tradingFees,
		minimumProfitThreshold: minimumProfitThreshold,
		entryPrice:             entryPrice,
		actualQuantityHeld:     actualQuantityHeld,
		useFixedQuantity:       useFixedQuantity,
		createdAt:              createdAt,
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

func (b *TradingBot) GetEntryPrice() float64 {
	return b.entryPrice
}

func (b *TradingBot) SetEntryPrice(price float64) {
	b.entryPrice = price
}

func (b *TradingBot) ClearEntryPrice() {
	b.entryPrice = 0.0
}

func (b *TradingBot) GetInitialCapital() float64 {
	return b.initialCapital
}

func (b *TradingBot) GetTradeAmount() float64 {
	return b.tradeAmount
}

func (b *TradingBot) GetCurrency() string {
	return b.currency
}

func (b *TradingBot) GetTradingFees() float64 {
	return b.tradingFees
}

func (b *TradingBot) GetMinimumProfitThreshold() float64 {
	return b.minimumProfitThreshold
}

func (b *TradingBot) GetActualQuantityHeld() float64 {
	return b.actualQuantityHeld
}

func (b *TradingBot) SetActualQuantityHeld(quantity float64) {
	b.actualQuantityHeld = quantity
}

func (b *TradingBot) ClearActualQuantityHeld() {
	b.actualQuantityHeld = 0.0
}

func (b *TradingBot) GetUseFixedQuantity() bool {
	return b.useFixedQuantity
}

func (b *TradingBot) SetUseFixedQuantity(useFixed bool) {
	b.useFixedQuantity = useFixed
}

// CalculateQuantityForSell calculates the quantity available for selling after considering trading fees
func (b *TradingBot) CalculateQuantityForSell() float64 {
	if b.actualQuantityHeld > 0 {
		// Use the actual quantity held (after buy fees)
		return b.actualQuantityHeld
	}
	// Fallback to original quantity minus estimated fees
	feePercentage := b.tradingFees / 100.0
	return b.quantity * (1.0 - feePercentage)
}

func (b *TradingBot) Stop() error {
	if b.status == StatusStopped {
		return fmt.Errorf("bot is already stopped")
	}
	b.status = StatusStopped
	return nil
}
