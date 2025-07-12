package service

import (
	"context"
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/infra/external"
	"fmt"
	"github.com/adshao/go-binance/v2"
	"strconv"
	"time"
)

// LiveTradingExecutionContext implements TradingExecutionContext for real trading
type LiveTradingExecutionContext struct {
	client                       external.BinanceClientInterface
	tradingBotRepository         repository.TradingBotRepository
	tradingDecisionLogRepository repository.TradingDecisionLogRepository
	shouldContinue               bool
}

// NewLiveTradingExecutionContext creates a new LiveTradingExecutionContext
func NewLiveTradingExecutionContext(
	client external.BinanceClientInterface,
	tradingBotRepo repository.TradingBotRepository,
	decisionLogRepo repository.TradingDecisionLogRepository,
) *LiveTradingExecutionContext {
	return &LiveTradingExecutionContext{
		client:                       client,
		tradingBotRepository:         tradingBotRepo,
		tradingDecisionLogRepository: decisionLogRepo,
		shouldContinue:               true,
	}
}

// ExecuteTrade executes real trading orders via Binance API
func (ctx *LiveTradingExecutionContext) ExecuteTrade(decision entity.TradingDecision, bot *entity.TradingBot, currentPrice float64, timestamp time.Time) error {
	symbol := bot.GetSymbol().GetValue()
	quantity := bot.GetQuantity()

	switch decision {
	case entity.Buy:
		if bot.GetIsPositioned() {
			return fmt.Errorf("this trading bot already has an open position")
		}
		fmt.Printf("🟢 Executing BUY order for %s, quantity: %.6f\n", symbol, quantity)

		isOrderPlaced := ctx.placeBuyOrder(symbol, quantity)
		if isOrderPlaced {
			// Set entry price when entering position
			bot.SetEntryPrice(currentPrice)
			fmt.Printf("📈 Entry price set to: %.2f for bot %s\n", currentPrice, bot.Id.GetValue())

			errPosition := bot.GetIntoPosition()
			if errPosition != nil {
				return errPosition
			}
			errUpdate := ctx.tradingBotRepository.Update(bot)
			if errUpdate != nil {
				return errUpdate
			}
		}
		return nil

	case entity.Sell:
		if !bot.GetIsPositioned() {
			return fmt.Errorf("this trading bot don't have an open position")
		}

		actualProfit := ((currentPrice - bot.GetEntryPrice()) / bot.GetEntryPrice()) * 100
		fmt.Printf("🔴 Executing SELL order for %s, quantity: %.6f (Profit: %.2f%%)\n", symbol, quantity, actualProfit)

		isOrderPlaced := ctx.placeSellOrder(symbol, quantity)
		if isOrderPlaced {
			// Clear entry price when exiting position
			bot.ClearEntryPrice()
			fmt.Printf("📉 Entry price cleared for bot %s after profitable sale\n", bot.Id.GetValue())

			errPosition := bot.GetOutOfPosition()
			if errPosition != nil {
				return errPosition
			}
			errUpdate := ctx.tradingBotRepository.Update(bot)
			if errUpdate != nil {
				return errUpdate
			}
		}
		return nil

	case entity.Hold:
		if bot.GetIsPositioned() {
			potentialProfit := ((currentPrice - bot.GetEntryPrice()) / bot.GetEntryPrice()) * 100
			fmt.Printf("⏸ HOLDING position for %s (Current potential profit: %.2f%%)\n", symbol, potentialProfit)
		} else {
			fmt.Printf("⏸ HOLDING (no position) for %s\n", symbol)
		}
	}

	return nil
}

// OnDecisionMade logs trading decisions to the repository
func (ctx *LiveTradingExecutionContext) OnDecisionMade(decisionLog *entity.TradingDecisionLog) error {
	if err := ctx.tradingDecisionLogRepository.Save(decisionLog); err != nil {
		fmt.Printf("⚠️ Failed to save decision log: %v\n", err)
		return err
	}
	return nil
}

// ShouldContinue returns whether the trading loop should continue
func (ctx *LiveTradingExecutionContext) ShouldContinue() bool {
	return ctx.shouldContinue
}

// Stop stops the trading execution context
func (ctx *LiveTradingExecutionContext) Stop() {
	ctx.shouldContinue = false
}

// placeBuyOrder places a real buy order via Binance API
func (ctx *LiveTradingExecutionContext) placeBuyOrder(symbol string, quantity float64) bool {
	qtyStr := strconv.FormatFloat(quantity, 'f', 6, 64)

	order, err := ctx.client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeBuy).
		Type(binance.OrderTypeMarket).
		Quantity(qtyStr).
		Do(context.Background())

	if err != nil {
		fmt.Printf("❌ Error placing buy order: %v\n", err)
		return false
	}

	fmt.Printf("✅ Buy order placed: %+v\n", order)
	return true
}

// placeSellOrder places a real sell order via Binance API
func (ctx *LiveTradingExecutionContext) placeSellOrder(symbol string, quantity float64) bool {
	qtyStr := strconv.FormatFloat(quantity, 'f', 6, 64)

	order, err := ctx.client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeSell).
		Type(binance.OrderTypeMarket).
		Quantity(qtyStr).
		Do(context.Background())

	if err != nil {
		fmt.Printf("❌ Error placing sell order: %v\n", err)
		return false
	}

	fmt.Printf("✅ Sell order placed: %+v\n", order)
	return true
}