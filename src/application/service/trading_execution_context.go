package service

import (
	"crypgo-machine/src/domain/entity"
	"time"
)

// TradingExecutionContext abstracts the execution environment for trading operations
type TradingExecutionContext interface {
	// ExecuteTrade executes a trading decision (buy, sell, or hold)
	ExecuteTrade(decision entity.TradingDecision, bot *entity.TradingBot, currentPrice float64, timestamp time.Time) error
	
	// OnDecisionMade is called when a trading decision is made (for logging/tracking)
	OnDecisionMade(decisionLog *entity.TradingDecisionLog) error
	
	// ShouldContinue returns whether the trading loop should continue
	ShouldContinue() bool
}