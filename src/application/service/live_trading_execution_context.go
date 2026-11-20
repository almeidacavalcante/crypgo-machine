package service

import (
	"context"
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/service"
	"crypgo-machine/src/infra/external"
	"crypgo-machine/src/infra/queue"
	"encoding/json"
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
	messageBroker                queue.MessageBroker
	orderValidator               *service.OrderValidatorService
	exchangeName                 string
	shouldContinue               bool
}

// NewLiveTradingExecutionContext creates a new LiveTradingExecutionContext
func NewLiveTradingExecutionContext(
	client external.BinanceClientInterface,
	tradingBotRepo repository.TradingBotRepository,
	decisionLogRepo repository.TradingDecisionLogRepository,
	messageBroker queue.MessageBroker,
	exchangeName string,
) *LiveTradingExecutionContext {
	// Create exchange info service and order validator
	exchangeInfoService := external.NewExchangeInfoService(client)
	orderValidator := service.NewOrderValidatorService(exchangeInfoService)

	return &LiveTradingExecutionContext{
		client:                       client,
		tradingBotRepository:         tradingBotRepo,
		tradingDecisionLogRepository: decisionLogRepo,
		messageBroker:                messageBroker,
		orderValidator:               orderValidator,
		exchangeName:                 exchangeName,
		shouldContinue:               true,
	}
}

// ExecuteTrade executes real trading orders via Binance API
func (ctx *LiveTradingExecutionContext) ExecuteTrade(decision entity.TradingDecision, bot *entity.TradingBot, currentPrice float64, timestamp time.Time) error {
	symbol := bot.GetSymbol().GetValue()

	// Calculate quantity based on useFixedQuantity flag
	var quantity float64
	if bot.GetUseFixedQuantity() {
		// Use fixed quantity from bot configuration
		quantity = bot.GetQuantity()
	} else {
		// Calculate dynamic quantity based on trade amount and current price
		quantity = bot.GetTradeAmount() / currentPrice
	}

	switch decision {
	case entity.Buy:
		if bot.GetIsPositioned() {
			return fmt.Errorf("this trading bot already has an open position")
		}
		fmt.Printf("🟢 [%s] BUY order (qty: %.6f, price: %.2f)\n", symbol, quantity, currentPrice)

		executedQuantity, isOrderPlaced := ctx.placeBuyOrder(symbol, quantity, currentPrice)
		if isOrderPlaced {
			// Set entry price when entering position
			bot.SetEntryPrice(currentPrice)
			// Calculate actual quantity held after fees from the quantity really
			// bought (after LOT_SIZE adjustment), not from the requested quantity
			feePercentage := bot.GetTradingFees() / 100.0
			actualQuantity := executedQuantity * (1.0 - feePercentage)
			bot.SetActualQuantityHeld(actualQuantity)
			fmt.Printf("📈 [%s] Position opened at %.2f (actual qty: %.6f after %.2f%% fees)\n",
				symbol, currentPrice, actualQuantity, bot.GetTradingFees())

			errPosition := bot.GetIntoPosition()
			if errPosition != nil {
				return errPosition
			}
			errUpdate := ctx.tradingBotRepository.Update(bot)
			if errUpdate != nil {
				return errUpdate
			}

			// Emit buy event
			if err := ctx.emitTradingEvent("trading.buy_executed", bot, currentPrice, executedQuantity, 0, 0, timestamp); err != nil {
				fmt.Printf("⚠️ Failed to emit buy event: %v\n", err)
			}
		}
		return nil

	case entity.Sell:
		if !bot.GetIsPositioned() {
			return fmt.Errorf("this trading bot don't have an open position")
		}

		// Calculate quantity for sell considering fees
		sellQuantity := bot.CalculateQuantityForSell()
		actualProfit := ((currentPrice - bot.GetEntryPrice()) / bot.GetEntryPrice()) * 100
		fmt.Printf("🔴 [%s] SELL order (qty: %.6f, profit: %.2f%%, entry: %.2f, current: %.2f)\n",
			symbol, sellQuantity, actualProfit, bot.GetEntryPrice(), currentPrice)

		executedSellQuantity, isOrderPlaced := ctx.placeSellOrder(symbol, sellQuantity, currentPrice)
		if isOrderPlaced {
			// Clear entry price and actual quantity when exiting position
			entryPrice := bot.GetEntryPrice()
			bot.ClearEntryPrice()
			bot.ClearActualQuantityHeld()
			fmt.Printf("📉 [%s] Position closed\n", symbol)

			errPosition := bot.GetOutOfPosition()
			if errPosition != nil {
				return errPosition
			}
			errUpdate := ctx.tradingBotRepository.Update(bot)
			if errUpdate != nil {
				return errUpdate
			}

			// Emit sell event
			if err := ctx.emitTradingEvent("trading.sell_executed", bot, currentPrice, executedSellQuantity, entryPrice, actualProfit, timestamp); err != nil {
				fmt.Printf("⚠️ Failed to emit sell event: %v\n", err)
			}
		}
		return nil

	case entity.Hold:
		if bot.GetIsPositioned() {
			entryPrice := bot.GetEntryPrice()
			if entryPrice > 0 {
				potentialProfit := ((currentPrice - entryPrice) / entryPrice) * 100
				// Only log if profit is significant or price has changed meaningfully
				if potentialProfit > 1.0 || potentialProfit < -1.0 {
					fmt.Printf("⏸ [%s] HOLDING position (profit: %.2f%%, entry: %.2f, current: %.2f)\n",
						symbol, potentialProfit, entryPrice, currentPrice)
				}
			}
		}
		// Remove the "no position" hold messages as they're too verbose
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

// resolveExecutedQuantity returns the base-asset quantity that was really traded.
// It prefers the exchange-reported executed quantity; when that is missing or not
// positive it falls back to the quantity string sent to Binance and then to the
// validator's adjusted quantity.
func resolveExecutedQuantity(order *binance.CreateOrderResponse, formattedQty string, adjustedQty float64) float64 {
	if order != nil {
		if executed, err := strconv.ParseFloat(order.ExecutedQuantity, 64); err == nil && executed > 0 {
			return executed
		}
	}
	if sent, err := strconv.ParseFloat(formattedQty, 64); err == nil && sent > 0 {
		return sent
	}
	return adjustedQty
}

// placeBuyOrder places a real buy order via Binance API with validation.
// It returns the executed quantity and whether the order was placed.
func (ctx *LiveTradingExecutionContext) placeBuyOrder(symbol string, quantity, price float64) (float64, bool) {
	// Validate and adjust quantity
	adjustedQty, formattedQty, shouldProceed, warnings := ctx.orderValidator.ValidateOrderBeforePlacement(symbol, quantity, price)

	if !shouldProceed {
		fmt.Printf("❌ Buy order validation failed for %s\n", symbol)
		return 0, false
	}

	// Log warnings if any
	for _, warning := range warnings {
		fmt.Printf("⚠️ [%s] %s\n", symbol, warning)
	}

	order, err := ctx.client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeBuy).
		Type(binance.OrderTypeMarket).
		Quantity(formattedQty).
		Do(context.Background())

	if err != nil {
		fmt.Printf("❌ Error placing buy order: %v\n", err)
		return 0, false
	}

	executedQty := resolveExecutedQuantity(order, formattedQty, adjustedQty)
	fmt.Printf("✅ Buy order placed: OrderID=%d, Qty=%s (adj: %.8f, executed: %.8f)\n", order.OrderID, formattedQty, adjustedQty, executedQty)
	return executedQty, true
}

// placeSellOrder places a real sell order via Binance API with validation.
// It returns the executed quantity and whether the order was placed.
func (ctx *LiveTradingExecutionContext) placeSellOrder(symbol string, quantity, price float64) (float64, bool) {
	// Validate and adjust quantity
	adjustedQty, formattedQty, shouldProceed, warnings := ctx.orderValidator.ValidateOrderBeforePlacement(symbol, quantity, price)

	if !shouldProceed {
		fmt.Printf("❌ Sell order validation failed for %s\n", symbol)
		return 0, false
	}

	// Log warnings if any
	for _, warning := range warnings {
		fmt.Printf("⚠️ [%s] %s\n", symbol, warning)
	}

	order, err := ctx.client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeSell).
		Type(binance.OrderTypeMarket).
		Quantity(formattedQty).
		Do(context.Background())

	if err != nil {
		fmt.Printf("❌ Error placing sell order: %v\n", err)
		return 0, false
	}

	executedQty := resolveExecutedQuantity(order, formattedQty, adjustedQty)
	fmt.Printf("✅ Sell order placed: OrderID=%d, Qty=%s (adj: %.8f, executed: %.8f)\n", order.OrderID, formattedQty, adjustedQty, executedQty)
	return executedQty, true
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
		"bot_id":       bot.Id.GetValue(),
		"symbol":       bot.GetSymbol().GetValue(),
		"action":       eventType[8:], // Remove "trading." prefix to get "buy_executed" or "sell_executed"
		"price":        price,
		"quantity":     quantity,
		"total_value":  totalValue,
		"strategy":     bot.GetStrategy().GetName(),
		"timestamp":    timestamp,
		"trading_fees": bot.GetTradingFees(),
		"currency":     bot.GetCurrency(),
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
		Headers: map[string]string{
			"timestamp": timestamp.Format(time.RFC3339),
			"bot_id":    bot.Id.GetValue(),
		},
	}

	return ctx.messageBroker.Publish(ctx.exchangeName, message)
}
