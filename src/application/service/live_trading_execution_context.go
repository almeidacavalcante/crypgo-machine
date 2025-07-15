package service

import (
	"context"
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/infra/external"
	"crypgo-machine/src/infra/queue"
	"encoding/json"
	"fmt"
	"github.com/adshao/go-binance/v2"
	"strconv"
	"sync"
	"time"
)

// LiveTradingExecutionContext implements TradingExecutionContext for real trading
type LiveTradingExecutionContext struct {
	client                       external.BinanceClientInterface
	tradingBotRepository         repository.TradingBotRepository
	tradingDecisionLogRepository repository.TradingDecisionLogRepository
	messageBroker                queue.MessageBroker
	exchangeName                 string
	shouldContinue               bool
	// Mutex to prevent race conditions during trade execution
	tradeMutex                   sync.RWMutex
}

// NewLiveTradingExecutionContext creates a new LiveTradingExecutionContext
func NewLiveTradingExecutionContext(
	client external.BinanceClientInterface,
	tradingBotRepo repository.TradingBotRepository,
	decisionLogRepo repository.TradingDecisionLogRepository,
	messageBroker queue.MessageBroker,
	exchangeName string,
) *LiveTradingExecutionContext {
	return &LiveTradingExecutionContext{
		client:                       client,
		tradingBotRepository:         tradingBotRepo,
		tradingDecisionLogRepository: decisionLogRepo,
		messageBroker:                messageBroker,
		exchangeName:                 exchangeName,
		shouldContinue:               true,
	}
}

// ExecuteTrade executes real trading orders via Binance API with race condition protection
func (ctx *LiveTradingExecutionContext) ExecuteTrade(decision entity.TradingDecision, bot *entity.TradingBot, currentPrice float64, timestamp time.Time) error {
	// Acquire exclusive lock to prevent race conditions
	ctx.tradeMutex.Lock()
	defer ctx.tradeMutex.Unlock()

	symbol := bot.GetSymbol().GetValue()
	quantity := bot.GetQuantity()
	botId := bot.Id.GetValue()

	fmt.Printf("🔒 [%s] Acquiring trade lock for %s decision (bot: %s)\n", symbol, decision, botId)

	// Get fresh bot state from database to avoid stale data
	freshBot, err := ctx.tradingBotRepository.GetTradeByID(botId)
	if err != nil {
		return fmt.Errorf("failed to get fresh bot state: %v", err)
	}

	switch decision {
	case entity.Buy:
		// Double-check with fresh bot state to prevent race conditions
		if freshBot.GetIsPositioned() {
			fmt.Printf("🚫 [%s] BUY rejected - bot already positioned (race condition prevented)\n", symbol)
			return fmt.Errorf("this trading bot already has an open position (fresh check)")
		}
		
		// Additional validation: check if bot was positioned in the last few seconds
		if bot.GetIsPositioned() {
			fmt.Printf("🚫 [%s] BUY rejected - bot positioned in memory (race condition prevented)\n", symbol)
			return fmt.Errorf("this trading bot already has an open position (memory check)")
		}
		
		fmt.Printf("🟢 [%s] BUY order validated (qty: %.6f, price: %.2f, bot: %s)\n", symbol, quantity, currentPrice, botId)

		isOrderPlaced := ctx.placeBuyOrder(symbol, quantity)
		if isOrderPlaced {
			// Set entry price when entering position
			freshBot.SetEntryPrice(currentPrice)
			fmt.Printf("📈 [%s] Position opened at %.2f (bot: %s)\n", symbol, currentPrice, botId)

			errPosition := freshBot.GetIntoPosition()
			if errPosition != nil {
				fmt.Printf("❌ [%s] Failed to set positioned state: %v\n", symbol, errPosition)
				return errPosition
			}
			
			// Critical: Update database immediately after position change
			errUpdate := ctx.tradingBotRepository.Update(freshBot)
			if errUpdate != nil {
				fmt.Printf("❌ [%s] Failed to update bot position in database: %v\n", symbol, errUpdate)
				return errUpdate
			}
			
			fmt.Printf("✅ [%s] Bot position updated in database (bot: %s)\n", symbol, botId)

			// Emit buy event
			if err := ctx.emitTradingEvent("trading.buy_executed", freshBot, currentPrice, quantity, 0, 0, timestamp); err != nil {
				fmt.Printf("⚠️ Failed to emit buy event: %v\n", err)
			}
			
			// Update the original bot reference to keep consistency
			bot.SetEntryPrice(currentPrice)
			errOriginalPosition := bot.GetIntoPosition()
			if errOriginalPosition != nil {
				fmt.Printf("⚠️ [%s] Failed to set original bot position: %v\n", symbol, errOriginalPosition)
				// Continue execution as the main operation (freshBot) already succeeded
			}
			
			// Save the updated original bot to database to maintain consistency
			errUpdateOriginal := ctx.tradingBotRepository.Update(bot)
			if errUpdateOriginal != nil {
				fmt.Printf("⚠️ [%s] Failed to update original bot reference in database: %v\n", symbol, errUpdateOriginal)
				// Note: We don't return error here because the main operation (freshBot) already succeeded
			}
		}
		return nil

	case entity.Sell:
		// Double-check with fresh bot state to prevent race conditions
		if !freshBot.GetIsPositioned() {
			fmt.Printf("🚫 [%s] SELL rejected - bot not positioned (race condition prevented)\n", symbol)
			return fmt.Errorf("this trading bot don't have an open position (fresh check)")
		}
		
		// Additional validation: check memory state
		if !bot.GetIsPositioned() {
			fmt.Printf("🚫 [%s] SELL rejected - bot not positioned in memory (race condition prevented)\n", symbol)
			return fmt.Errorf("this trading bot don't have an open position (memory check)")
		}

		actualProfit := ((currentPrice - freshBot.GetEntryPrice()) / freshBot.GetEntryPrice()) * 100
		fmt.Printf("🔴 [%s] SELL order validated (qty: %.6f, profit: %.2f%%, entry: %.2f, current: %.2f, bot: %s)\n", 
			symbol, quantity, actualProfit, freshBot.GetEntryPrice(), currentPrice, botId)

		isOrderPlaced := ctx.placeSellOrder(symbol, quantity)
		if isOrderPlaced {
			// Clear entry price when exiting position
			entryPrice := freshBot.GetEntryPrice()
			freshBot.ClearEntryPrice()
			fmt.Printf("📉 [%s] Position closed (bot: %s)\n", symbol, botId)

			errPosition := freshBot.GetOutOfPosition()
			if errPosition != nil {
				fmt.Printf("❌ [%s] Failed to clear positioned state: %v\n", symbol, errPosition)
				return errPosition
			}
			
			// Critical: Update database immediately after position change
			errUpdate := ctx.tradingBotRepository.Update(freshBot)
			if errUpdate != nil {
				fmt.Printf("❌ [%s] Failed to update bot position in database: %v\n", symbol, errUpdate)
				return errUpdate
			}
			
			fmt.Printf("✅ [%s] Bot position cleared in database (bot: %s)\n", symbol, botId)

			// Emit sell event
			if err := ctx.emitTradingEvent("trading.sell_executed", freshBot, currentPrice, quantity, entryPrice, actualProfit, timestamp); err != nil {
				fmt.Printf("⚠️ Failed to emit sell event: %v\n", err)
			}
			
			// Update the original bot reference to keep consistency
			bot.ClearEntryPrice()
			errOriginalPosition := bot.GetOutOfPosition()
			if errOriginalPosition != nil {
				fmt.Printf("⚠️ [%s] Failed to clear original bot position: %v\n", symbol, errOriginalPosition)
				// Continue execution as the main operation (freshBot) already succeeded
			}
			
			// Save the updated original bot to database to maintain consistency
			errUpdateOriginal := ctx.tradingBotRepository.Update(bot)
			if errUpdateOriginal != nil {
				fmt.Printf("⚠️ [%s] Failed to update original bot reference in database: %v\n", symbol, errUpdateOriginal)
				// Note: We don't return error here because the main operation (freshBot) already succeeded
			}
		}
		return nil

	case entity.Hold:
		if bot.GetIsPositioned() {
			entryPrice := bot.GetEntryPrice()
			if entryPrice > 0 {
				potentialProfit := ((currentPrice - entryPrice) / entryPrice) * 100
				fmt.Printf("⏸ [%s] HOLDING position (profit: %.2f%%, entry: %.2f, current: %.2f)\n", 
					symbol, potentialProfit, entryPrice, currentPrice)
			} else {
				fmt.Printf("⏸ [%s] HOLDING position (entry price unavailable)\n", symbol)
			}
		} else {
			fmt.Printf("⏸ [%s] HOLDING (no position)\n", symbol)
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

// emitTradingEvent emits trading events to the message broker
func (ctx *LiveTradingExecutionContext) emitTradingEvent(
	eventType string,
	bot *entity.TradingBot,
	price float64,
	quantity float64,
	entryPrice float64,
	profitLoss float64,
	timestamp time.Time,
) error {
	totalValue := price * quantity
	
	payload := map[string]interface{}{
		"bot_id":           bot.Id.GetValue(),
		"symbol":           bot.GetSymbol().GetValue(),
		"action":           eventType[8:], // Remove "trading." prefix to get "buy_executed" or "sell_executed"
		"price":            price,
		"quantity":         quantity,
		"total_value":      totalValue,
		"strategy":         bot.GetStrategy().GetName(),
		"timestamp":        timestamp,
		"trading_fees":     bot.GetTradingFees(),
		"currency":         bot.GetCurrency(),
	}

	// Add extra fields for sell events
	if eventType == "trading.sell_executed" {
		payload["entry_price"] = entryPrice
		payload["profit_loss"] = (price - entryPrice) * quantity
		payload["profit_loss_perc"] = profitLoss
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal trading event payload: %v", err)
	}

	message := queue.Message{
		RoutingKey: eventType,
		Payload:    payloadBytes,
		Headers:    map[string]string{
			"timestamp": timestamp.Format(time.RFC3339),
			"bot_id":    bot.Id.GetValue(),
		},
	}

	return ctx.messageBroker.Publish(ctx.exchangeName, message)
}