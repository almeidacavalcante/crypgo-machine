package service

import (
	"crypgo-machine/src/domain/entity"
	"fmt"
	"time"
)

// BacktestResult holds the results of a backtest
type BacktestResult struct {
	Symbol             string                           `json:"symbol"`
	InitialCapital     float64                          `json:"initial_capital"`
	FinalCapital       float64                          `json:"final_capital"`
	TotalPnL           float64                          `json:"total_pnl"`
	ROI                float64                          `json:"roi"`
	WinRate            float64                          `json:"win_rate"`
	TotalTrades        int                              `json:"total_trades"`
	WinningTrades      int                              `json:"winning_trades"`
	LosingTrades       int                              `json:"losing_trades"`
	MaxDrawdown        float64                          `json:"max_drawdown"`
	TradingFees        float64                          `json:"trading_fees"`
	Decisions          []*entity.TradingDecisionLog     `json:"decisions"`
	Trades             []BacktestTrade                  `json:"trades"`
}

// BacktestTrade represents a completed trade in the backtest
type BacktestTrade struct {
	EntryPrice    float64   `json:"entry_price"`
	ExitPrice     float64   `json:"exit_price"`
	EntryTime     time.Time `json:"entry_time"`
	ExitTime      time.Time `json:"exit_time"`
	Quantity      float64   `json:"quantity"`
	PnL           float64   `json:"pnl"`
	PnLPercentage float64   `json:"pnl_percentage"`
	Fees          float64   `json:"fees"`
}

// BacktestTradingExecutionContext implements TradingExecutionContext for backtesting
type BacktestTradingExecutionContext struct {
	result            *BacktestResult
	currentTrade      *BacktestTrade
	shouldContinue    bool
	highWaterMark     float64 // For drawdown calculation
}

// NewBacktestTradingExecutionContext creates a new BacktestTradingExecutionContext
func NewBacktestTradingExecutionContext(symbol string, initialCapital float64) *BacktestTradingExecutionContext {
	return &BacktestTradingExecutionContext{
		result: &BacktestResult{
			Symbol:         symbol,
			InitialCapital: initialCapital,
			FinalCapital:   initialCapital,
			Decisions:      make([]*entity.TradingDecisionLog, 0),
			Trades:         make([]BacktestTrade, 0),
		},
		shouldContinue: true,
		highWaterMark:  initialCapital,
	}
}

// ExecuteTrade simulates trading operations and updates backtest metrics
func (ctx *BacktestTradingExecutionContext) ExecuteTrade(decision entity.TradingDecision, bot *entity.TradingBot, currentPrice float64, timestamp time.Time) error {
	switch decision {
	case entity.Buy:
		if bot.GetIsPositioned() {
			return fmt.Errorf("bot already has an open position")
		}

		// Simulate buy order
		fmt.Printf("🟢 [BACKTEST] BUY at %.2f on %s\n", currentPrice, timestamp.Format("2006-01-02 15:04"))
		
		// Calculate fees
		tradeValue := bot.GetTradeAmount()
		fees := tradeValue * (bot.GetTradingFees() / 100)
		
		// Start a new trade
		ctx.currentTrade = &BacktestTrade{
			EntryPrice: currentPrice,
			EntryTime:  timestamp,
			Quantity:   bot.GetQuantity(),
			Fees:       fees,
		}
		
		// Update bot state
		bot.SetEntryPrice(currentPrice)
		_ = bot.GetIntoPosition()
		
		// Update capital (deduct fees)
		ctx.result.FinalCapital -= fees
		ctx.result.TradingFees += fees

	case entity.Sell:
		if !bot.GetIsPositioned() {
			return fmt.Errorf("bot has no open position")
		}
		
		if ctx.currentTrade == nil {
			return fmt.Errorf("no current trade to close")
		}

		// Calculate profit/loss
		entryPrice := bot.GetEntryPrice()
		tradeValue := bot.GetTradeAmount()
		fees := tradeValue * (bot.GetTradingFees() / 100)
		
		pnlPercentage := ((currentPrice - entryPrice) / entryPrice) * 100
		pnl := (tradeValue * pnlPercentage / 100) - fees // Subtract exit fees
		
		// Complete the trade
		ctx.currentTrade.ExitPrice = currentPrice
		ctx.currentTrade.ExitTime = timestamp
		ctx.currentTrade.PnL = pnl
		ctx.currentTrade.PnLPercentage = pnlPercentage
		ctx.currentTrade.Fees += fees // Add exit fees
		
		// Update metrics
		ctx.result.Trades = append(ctx.result.Trades, *ctx.currentTrade)
		ctx.result.TotalTrades++
		ctx.result.TotalPnL += pnl
		ctx.result.FinalCapital += pnl
		ctx.result.TradingFees += fees
		
		if pnl > 0 {
			ctx.result.WinningTrades++
		} else {
			ctx.result.LosingTrades++
		}
		
		// Update high water mark and check for drawdown
		if ctx.result.FinalCapital > ctx.highWaterMark {
			ctx.highWaterMark = ctx.result.FinalCapital
		} else {
			drawdown := ((ctx.highWaterMark - ctx.result.FinalCapital) / ctx.highWaterMark) * 100
			if drawdown > ctx.result.MaxDrawdown {
				ctx.result.MaxDrawdown = drawdown
			}
		}
		
		fmt.Printf("🔴 [BACKTEST] SELL at %.2f on %s (P&L: %.2f BRL, %.2f%%)\n", 
			currentPrice, timestamp.Format("2006-01-02 15:04"), pnl, pnlPercentage)
		
		// Update bot state
		bot.ClearEntryPrice()
		_ = bot.GetOutOfPosition()
		ctx.currentTrade = nil

	case entity.Hold:
		if bot.GetIsPositioned() {
			potentialProfit := ((currentPrice - bot.GetEntryPrice()) / bot.GetEntryPrice()) * 100
			fmt.Printf("⏸ [BACKTEST] HOLDING at %.2f (Potential profit: %.2f%%)\n", currentPrice, potentialProfit)
		} else {
			fmt.Printf("⏸ [BACKTEST] HOLDING (no position) at %.2f\n", currentPrice)
		}
	}

	return nil
}

// OnDecisionMade stores the decision for analysis
func (ctx *BacktestTradingExecutionContext) OnDecisionMade(decisionLog *entity.TradingDecisionLog) error {
	ctx.result.Decisions = append(ctx.result.Decisions, decisionLog)
	return nil
}

// ShouldContinue returns whether the backtest should continue
func (ctx *BacktestTradingExecutionContext) ShouldContinue() bool {
	return ctx.shouldContinue
}

// Stop stops the backtest
func (ctx *BacktestTradingExecutionContext) Stop() {
	ctx.shouldContinue = false
}

// GetResult finalizes and returns the backtest results
func (ctx *BacktestTradingExecutionContext) GetResult() *BacktestResult {
	// Calculate final metrics
	if ctx.result.TotalTrades > 0 {
		ctx.result.WinRate = (float64(ctx.result.WinningTrades) / float64(ctx.result.TotalTrades)) * 100
	}
	
	if ctx.result.InitialCapital > 0 {
		ctx.result.ROI = ((ctx.result.FinalCapital - ctx.result.InitialCapital) / ctx.result.InitialCapital) * 100
	}
	
	return ctx.result
}