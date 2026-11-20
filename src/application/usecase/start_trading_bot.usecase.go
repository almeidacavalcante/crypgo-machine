package usecase

import (
	"context"
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/application/service"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"crypgo-machine/src/infra/queue"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type StartTradingBotUseCase struct {
	tradingBotRepository         repository.TradingBotRepository
	tradingDecisionLogRepository repository.TradingDecisionLogRepository
	client                       external.BinanceClientInterface
	dataSource                   service.MarketDataSource
	executionContext             service.TradingExecutionContext
	messageBroker                queue.MessageBroker
	exchangeName                 string
}

func NewStartTradingBotUseCase(
	tradingBotRepo repository.TradingBotRepository,
	decisionLogRepo repository.TradingDecisionLogRepository,
	client external.BinanceClientInterface,
) *StartTradingBotUseCase {
	// Create default live implementations for backward compatibility
	dataSource := service.NewLiveMarketDataSource(client)
	executionContext := service.NewLiveTradingExecutionContext(client, tradingBotRepo, decisionLogRepo, nil, "")

	return &StartTradingBotUseCase{
		tradingBotRepository:         tradingBotRepo,
		tradingDecisionLogRepository: decisionLogRepo,
		client:                       client,
		dataSource:                   dataSource,
		executionContext:             executionContext,
	}
}

// NewStartTradingBotUseCaseWithMessaging creates a new StartTradingBotUseCase with message broker for notifications
func NewStartTradingBotUseCaseWithMessaging(
	tradingBotRepo repository.TradingBotRepository,
	decisionLogRepo repository.TradingDecisionLogRepository,
	client external.BinanceClientInterface,
	messageBroker queue.MessageBroker,
	exchangeName string,
) *StartTradingBotUseCase {
	// Create live implementations with messaging support
	dataSource := service.NewLiveMarketDataSource(client)
	executionContext := service.NewLiveTradingExecutionContext(client, tradingBotRepo, decisionLogRepo, messageBroker, exchangeName)

	return &StartTradingBotUseCase{
		tradingBotRepository:         tradingBotRepo,
		tradingDecisionLogRepository: decisionLogRepo,
		client:                       client,
		dataSource:                   dataSource,
		executionContext:             executionContext,
		messageBroker:                messageBroker,
		exchangeName:                 exchangeName,
	}
}

// NewStartTradingBotUseCaseWithServices creates a new StartTradingBotUseCase with custom services
func NewStartTradingBotUseCaseWithServices(
	tradingBotRepo repository.TradingBotRepository,
	decisionLogRepo repository.TradingDecisionLogRepository,
	client external.BinanceClientInterface,
	dataSource service.MarketDataSource,
	executionContext service.TradingExecutionContext,
) *StartTradingBotUseCase {
	return &StartTradingBotUseCase{
		tradingBotRepository:         tradingBotRepo,
		tradingDecisionLogRepository: decisionLogRepo,
		client:                       client,
		dataSource:                   dataSource,
		executionContext:             executionContext,
	}
}

type InputStartTradingBot struct {
	TradingBotId string `json:"bot_id"`
	// SkipNotification suppresses the "trading_bot.started" event (e.g. for auto-recovery on boot).
	// It is never read from the request body.
	SkipNotification bool `json:"-"`
}

func (uc *StartTradingBotUseCase) Execute(input InputStartTradingBot) error {
	tradingBot, err := uc.tradingBotRepository.GetTradeByID(input.TradingBotId)
	if err != nil {
		return err
	}
	if tradingBot == nil {
		return fmt.Errorf("trading bot not found")
	}

	// Reject bots whose interval the exchange does not support before marking them RUNNING:
	// an invalid interval would make the loop fail on every tick or panic in time.NewTicker.
	if _, errInterval := external.SecondsToInterval(tradingBot.GetIntervalSeconds()); errInterval != nil {
		return fmt.Errorf("cannot start trading bot %s: invalid interval_seconds: %w", tradingBot.Id.GetValue(), errInterval)
	}

	errStart := tradingBot.Start()
	if errStart != nil {
		return errStart
	}

	errSave := uc.tradingBotRepository.Update(tradingBot)
	if errSave != nil {
		return errSave
	}

	if !input.SkipNotification {
		uc.publishBotEvent("trading_bot.started", tradingBot, "")
	}

	go uc.runStrategyLoop(tradingBot)

	fmt.Printf("✅ [%s] Bot started - %s strategy (%.6f qty, %ds intervals)\n",
		tradingBot.GetSymbol().GetValue(),
		tradingBot.GetStrategy().GetName(),
		tradingBot.GetQuantity(),
		tradingBot.GetIntervalSeconds())

	return nil
}

// publishBotEvent publishes a bot lifecycle event. Failures are logged and never returned,
// so a broker outage cannot fail a start or stop.
func (uc *StartTradingBotUseCase) publishBotEvent(routingKey string, tradingBot *entity.TradingBot, reason string) {
	publishTradingBotLifecycleEvent(uc.messageBroker, uc.exchangeName, routingKey, tradingBot, reason)
}

// publishTradingBotLifecycleEvent publishes "trading_bot.started" / "trading_bot.stopped" events
// with the fields the email and Telegram consumers read ("id", "symbol").
func publishTradingBotLifecycleEvent(messageBroker queue.MessageBroker, exchangeName string, routingKey string, tradingBot *entity.TradingBot, reason string) {
	if messageBroker == nil || tradingBot == nil {
		return
	}

	data := map[string]interface{}{
		"id":       tradingBot.Id.GetValue(),
		"symbol":   tradingBot.GetSymbol().GetValue(),
		"status":   string(tradingBot.GetStatus()),
		"strategy": tradingBot.GetStrategy().GetName(),
	}
	if reason != "" {
		data["reason"] = reason
	}

	payload, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("⚠️ Failed to marshal %s event for bot %s: %v\n", routingKey, tradingBot.Id.GetValue(), err)
		return
	}

	event := queue.Message{
		RoutingKey: routingKey,
		Payload:    payload,
		Headers: map[string]string{
			"event_type": routingKey,
		},
	}
	if err := messageBroker.Publish(exchangeName, event); err != nil {
		fmt.Printf("⚠️ Failed to publish %s event for bot %s: %v\n", routingKey, tradingBot.Id.GetValue(), err)
	}
}

// recoverStrategyLoopPanic stops a bot whose strategy loop panicked, so the panic cannot kill
// the whole process and the bot is not restarted by auto-recovery on the next boot.
// It must be called directly by a deferred call in the loop goroutine.
func (uc *StartTradingBotUseCase) recoverStrategyLoopPanic(tradingBot *entity.TradingBot) {
	r := recover()
	if r == nil {
		return
	}

	botId := tradingBot.Id.GetValue()
	fmt.Printf("🚨 Trading bot %s strategy loop panicked: %v\n", botId, r)

	// Persisting the stopped status must never panic out of this goroutine either.
	defer func() {
		if r2 := recover(); r2 != nil {
			fmt.Printf("🚨 Failed to persist stopped status for trading bot %s after panic: %v\n", botId, r2)
		}
	}()

	if uc.tradingBotRepository == nil {
		return
	}

	// Prefer the persisted copy so concurrent changes (e.g. a stop request) are not overwritten.
	botToStop := tradingBot
	if currentBot, err := uc.tradingBotRepository.GetTradeByID(botId); err == nil && currentBot != nil {
		botToStop = currentBot
	}

	if botToStop.GetStatus() == entity.StatusStopped {
		return
	}

	// The entity exposes no way to set StatusError, so the bot is marked STOPPED.
	if err := botToStop.Stop(); err != nil {
		fmt.Printf("🚨 Failed to stop trading bot %s after panic: %v\n", botId, err)
		return
	}
	if err := uc.tradingBotRepository.Update(botToStop); err != nil {
		fmt.Printf("🚨 Failed to persist stopped status for trading bot %s after panic: %v\n", botId, err)
		return
	}

	fmt.Printf("🛑 Trading bot %s marked as STOPPED after strategy loop panic\n", botId)
	uc.publishBotEvent("trading_bot.stopped", botToStop, fmt.Sprintf("strategy loop panic: %v", r))
}

func (uc *StartTradingBotUseCase) runStrategyLoop(tradingBot *entity.TradingBot) {
	defer uc.recoverStrategyLoopPanic(tradingBot)

	// Execute first analysis immediately
	shouldContinue := uc.executeAnalysisAndTrade(tradingBot)
	if !shouldContinue {
		fmt.Printf("🛑 Trading bot %s stopped before starting loop\n", tradingBot.Id.GetValue())
		return
	}

	// Then start the ticker for subsequent executions
	ticker := time.NewTicker(time.Duration(tradingBot.GetIntervalSeconds()) * time.Second)
	defer ticker.Stop()

	// Status ticker - show summary every 10 minutes
	statusTicker := time.NewTicker(10 * time.Minute)
	defer statusTicker.Stop()

	for {
		select {
		case <-ticker.C:
			shouldContinue := uc.executeAnalysisAndTrade(tradingBot)
			if !shouldContinue {
				fmt.Printf("🛑 Trading bot %s stopped, exiting loop\n", tradingBot.Id.GetValue())
				return // Exit the loop completely
			}
		case <-statusTicker.C:
			// Show periodic status summary
			symbol := tradingBot.GetSymbol().GetValue()
			if tradingBot.GetIsPositioned() {
				entryPrice := tradingBot.GetEntryPrice()
				if entryPrice > 0 {
					// Get current price for profit calculation
					klines, err := uc.dataSource.GetMarketData(symbol, tradingBot.GetIntervalSeconds())
					if err == nil && len(klines) > 0 {
						currentPrice := klines[len(klines)-1].Close()
						profit := ((currentPrice - entryPrice) / entryPrice) * 100
						fmt.Printf("📊 [%s] Status: POSITIONED (entry: %.2f, current: %.2f, profit: %.2f%%)\n",
							symbol, entryPrice, currentPrice, profit)
					}
				}
			} else {
				fmt.Printf("📊 [%s] Status: MONITORING (no position)\n", symbol)
			}
		}
	}
}

// ExecuteAnalysisAndTrade performs a single analysis and trading decision
// This method is public so it can be reused by other use cases like backtest
func (uc *StartTradingBotUseCase) ExecuteAnalysisAndTrade(tradingBot *entity.TradingBot) error {
	// Fetch market data using abstraction with bot's configured interval
	klines, err := uc.dataSource.GetMarketData(tradingBot.GetSymbol().GetValue(), tradingBot.GetIntervalSeconds())
	if err != nil {
		return fmt.Errorf("error fetching market data for %s with interval %ds: %v", tradingBot.GetSymbol().GetValue(), tradingBot.GetIntervalSeconds(), err)
	}

	strategy := tradingBot.GetStrategy()
	analysisResult := strategy.Decide(klines, tradingBot)

	// Create and save decision log
	currentPrice := klines[len(klines)-1].Close()
	currentTime := uc.dataSource.GetCurrentTime()

	// Extract possible profit from analysis data, defaulting to 0.0 if not found
	possibleProfit := 0.0
	if profit, exists := analysisResult.AnalysisData["possibleProfit"]; exists {
		if profitFloat, ok := profit.(float64); ok {
			possibleProfit = profitFloat
		}
	}

	decisionLog := entity.NewTradingDecisionLog(
		tradingBot.Id,
		analysisResult.Decision,
		strategy.GetName(),
		analysisResult.AnalysisData,
		klines,
		currentPrice,
		possibleProfit,
	)

	// Save decision log using execution context
	if err := uc.executionContext.OnDecisionMade(decisionLog); err != nil {
		fmt.Printf("⚠️ Failed to save decision log: %v\n", err)
	}

	// Format analysis data in a more readable way
	symbol := tradingBot.GetSymbol().GetValue()

	// Only log significant events (no verbose HOLD messages)
	if analysisResult.Decision != entity.Hold {
		var analysisStr string
		if reason, exists := analysisResult.AnalysisData["reason"]; exists {
			analysisStr = fmt.Sprintf("reason: %v", reason)

			// Add key metrics if available
			if currentPrice, exists := analysisResult.AnalysisData["currentPrice"]; exists {
				analysisStr += fmt.Sprintf(", price: %.2f", currentPrice)
			}
			if fast, exists := analysisResult.AnalysisData["fast"]; exists {
				if slow, exists := analysisResult.AnalysisData["slow"]; exists {
					analysisStr += fmt.Sprintf(", MA(%.1f/%.1f)", fast, slow)
				}
			}
			if spread, exists := analysisResult.AnalysisData["actualSpread"]; exists {
				analysisStr += fmt.Sprintf(", spread: %.2f%%", spread)
			}
			if minSpread, exists := analysisResult.AnalysisData["minimumSpread"]; exists {
				analysisStr += fmt.Sprintf(" (min: %.1f%%)", minSpread)
			}
		} else {
			analysisStr = fmt.Sprintf("%+v", analysisResult.AnalysisData)
		}

		fmt.Printf("🤖 [%s] %s → %s (%s)\n",
			symbol, strategy.GetName(), analysisResult.Decision, analysisStr)
	}

	// Execute trading decision using abstraction
	if err := uc.executionContext.ExecuteTrade(analysisResult.Decision, tradingBot, currentPrice, currentTime); err != nil {
		return fmt.Errorf("error executing trade: %v", err)
	}

	return nil
}

func (uc *StartTradingBotUseCase) executeAnalysisAndTrade(tradingBot *entity.TradingBot) bool {
	currentBot, err := uc.tradingBotRepository.GetTradeByID(tradingBot.Id.GetValue())
	if err != nil || currentBot == nil || currentBot.GetStatus() != entity.StatusRunning {
		fmt.Printf("🔍 Trading bot %s status check: stopped or error (%v)\n", tradingBot.Id.GetValue(), err)
		return false // Stop the loop
	}

	// Check if execution context wants to continue
	if !uc.executionContext.ShouldContinue() {
		fmt.Printf("🔍 Trading bot %s execution context requested stop\n", tradingBot.Id.GetValue())
		return false // Stop the loop
	}

	// Use the public method to perform the analysis and trade
	if err := uc.ExecuteAnalysisAndTrade(tradingBot); err != nil {
		fmt.Printf("❌ Error in analysis and trade for bot %s: %v\n", tradingBot.Id.GetValue(), err)
		// Continue despite errors - don't stop the bot for temporary issues
	}

	return true // Continue the loop
}

// convertIntervalSecondsToBinanceInterval converts interval_seconds to Binance API interval format
func convertIntervalSecondsToBinanceInterval(intervalSeconds int) string {
	switch intervalSeconds {
	case 60:
		return "1m"
	case 180:
		return "3m"
	case 300:
		return "5m"
	case 900:
		return "15m"
	case 1800:
		return "30m"
	case 3600:
		return "1h"
	case 7200:
		return "2h"
	case 14400:
		return "4h"
	case 21600:
		return "6h"
	case 28800:
		return "8h"
	case 43200:
		return "12h"
	case 86400:
		return "1d"
	case 259200:
		return "3d"
	case 604800:
		return "1w"
	default:
		// Fallback: try to calculate the best match
		if intervalSeconds < 3600 {
			// Less than 1 hour, use minutes
			minutes := intervalSeconds / 60
			return fmt.Sprintf("%dm", minutes)
		} else if intervalSeconds < 86400 {
			// Less than 1 day, use hours
			hours := intervalSeconds / 3600
			return fmt.Sprintf("%dh", hours)
		} else {
			// Default to 1 hour for safety
			return "1h"
		}
	}
}

func (uc *StartTradingBotUseCase) getMarketData(symbol string, intervalSeconds int) ([]vo.Kline, error) {
	binanceInterval := convertIntervalSecondsToBinanceInterval(intervalSeconds)
	binanceKlines, err := uc.client.NewKlinesService().
		Symbol(symbol).
		Interval(binanceInterval).
		Limit(100).
		Do(context.Background())

	if err != nil {
		return nil, err
	}

	klines := make([]vo.Kline, len(binanceKlines))
	for i, bkline := range binanceKlines {
		openPrice, _ := strconv.ParseFloat(bkline.Open, 64)
		closePrice, _ := strconv.ParseFloat(bkline.Close, 64)
		highPrice, _ := strconv.ParseFloat(bkline.High, 64)
		lowPrice, _ := strconv.ParseFloat(bkline.Low, 64)
		volumePrice, _ := strconv.ParseFloat(bkline.Volume, 64)

		kline, err := vo.NewKline(openPrice, closePrice, highPrice, lowPrice, volumePrice, bkline.CloseTime)
		if err != nil {
			return nil, err
		}
		klines[i] = kline
	}

	return klines, nil
}
