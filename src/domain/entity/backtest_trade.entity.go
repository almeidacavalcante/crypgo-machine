package entity

import (
	"crypgo-machine/src/domain/vo"
	"fmt"
	"time"
)

type BacktestTrade struct {
	id          *vo.EntityId
	symbol      vo.Symbol
	decision    TradingDecision
	entryPrice  float64
	exitPrice   float64
	quantity    float64
	entryTime   time.Time
	exitTime    *time.Time
	profitLoss  *vo.ProfitLoss
	isOpen      bool
	reason      string
}

func NewBacktestTrade(
	symbol vo.Symbol,
	decision TradingDecision,
	price float64,
	quantity float64,
	entryTime time.Time,
	reason string,
) *BacktestTrade {
	id := vo.NewEntityId()
	
	return &BacktestTrade{
		id:         id,
		symbol:     symbol,
		decision:   decision,
		entryPrice: price,
		quantity:   quantity,
		entryTime:  entryTime,
		isOpen:     true,
		reason:     reason,
	}
}

func (bt *BacktestTrade) Close(exitPrice float64, exitTime time.Time, currency *vo.Currency) error {
	if !bt.isOpen {
		return fmt.Errorf("trade is already closed")
	}
	
	bt.exitPrice = exitPrice
	bt.exitTime = &exitTime
	bt.isOpen = false
	
	// Calculate P&L
	var plValue float64
	if bt.decision == Buy {
		// For buy: profit when exit price > entry price
		plValue = (exitPrice - bt.entryPrice) * bt.quantity
	} else {
		// For sell: profit when entry price > exit price  
		plValue = (bt.entryPrice - exitPrice) * bt.quantity
	}
	
	profitLoss, err := vo.NewProfitLoss(plValue, currency)
	if err != nil {
		return err
	}
	
	bt.profitLoss = &profitLoss
	return nil
}

func (bt *BacktestTrade) GetId() *vo.EntityId {
	return bt.id
}

func (bt *BacktestTrade) GetSymbol() vo.Symbol {
	return bt.symbol
}

func (bt *BacktestTrade) GetDecision() TradingDecision {
	return bt.decision
}

func (bt *BacktestTrade) GetEntryPrice() float64 {
	return bt.entryPrice
}

func (bt *BacktestTrade) GetExitPrice() float64 {
	return bt.exitPrice
}

func (bt *BacktestTrade) GetQuantity() float64 {
	return bt.quantity
}

func (bt *BacktestTrade) GetEntryTime() time.Time {
	return bt.entryTime
}

func (bt *BacktestTrade) GetExitTime() *time.Time {
	return bt.exitTime
}

func (bt *BacktestTrade) GetProfitLoss() *vo.ProfitLoss {
	return bt.profitLoss
}

func (bt *BacktestTrade) IsOpen() bool {
	return bt.isOpen
}

func (bt *BacktestTrade) GetReason() string {
	return bt.reason
}

func (bt *BacktestTrade) IsWinning() bool {
	if bt.profitLoss == nil {
		return false
	}
	return bt.profitLoss.IsProfit()
}