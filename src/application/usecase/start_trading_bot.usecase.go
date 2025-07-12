package usecase

import (
	"context"
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/application/service"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"fmt"
	"github.com/adshao/go-binance/v2"
	"strconv"
	"time"
)

type StartTradingBotUseCase struct {
	tradingBotRepository         repository.TradingBotRepository
	tradingDecisionLogRepository repository.TradingDecisionLogRepository
	client                       external.BinanceClientInterface
	dataSource                   service.MarketDataSource
	executionContext             service.TradingExecutionContext
}

func NewStartTradingBotUseCase(
	tradingBotRepo repository.TradingBotRepository,
	decisionLogRepo repository.TradingDecisionLogRepository,
	client external.BinanceClientInterface,
) *StartTradingBotUseCase {
	// Create default live implementations for backward compatibility
	dataSource := service.NewLiveMarketDataSource(client)
	executionContext := service.NewLiveTradingExecutionContext(client, tradingBotRepo, decisionLogRepo)
	
	return &StartTradingBotUseCase{
		tradingBotRepository:         tradingBotRepo,
		tradingDecisionLogRepository: decisionLogRepo,
		client:                       client,
		dataSource:                   dataSource,
		executionContext:             executionContext,
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
}

func (uc *StartTradingBotUseCase) Execute(input InputStartTradingBot) error {
	tradingBot, err := uc.tradingBotRepository.GetTradeByID(input.TradingBotId)
	if err != nil {
		return err
	}
	if tradingBot == nil {
		return fmt.Errorf("trading bot not found")
	}

	errStart := tradingBot.Start()
	if errStart != nil {
		return errStart
	}

	errSave := uc.tradingBotRepository.Update(tradingBot)
	if errSave != nil {
		return errSave
	}

	go uc.runStrategyLoop(tradingBot)

	return nil
}

func (uc *StartTradingBotUseCase) runStrategyLoop(tradingBot *entity.TradingBot) {
	// Execute first analysis immediately
	uc.executeAnalysisAndTrade(tradingBot)

	// Then start the ticker for subsequent executions
	ticker := time.NewTicker(time.Duration(tradingBot.GetIntervalSeconds()) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		uc.executeAnalysisAndTrade(tradingBot)
	}
}

// ExecuteAnalysisAndTrade performs a single analysis and trading decision
// This method is public so it can be reused by other use cases like backtest
func (uc *StartTradingBotUseCase) ExecuteAnalysisAndTrade(tradingBot *entity.TradingBot) error {
	// Fetch market data using abstraction
	klines, err := uc.dataSource.GetMarketData(tradingBot.GetSymbol().GetValue())
	if err != nil {
		return fmt.Errorf("error fetching market data for %s: %v", tradingBot.GetSymbol().GetValue(), err)
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

	fmt.Printf("🤖 Strategy %s for bot %s decided: %s (analysis: %+v)\n",
		strategy.GetName(), tradingBot.Id.GetValue(), analysisResult.Decision, analysisResult.AnalysisData)

	// Execute trading decision using abstraction
	if err := uc.executionContext.ExecuteTrade(analysisResult.Decision, tradingBot, currentPrice, currentTime); err != nil {
		return fmt.Errorf("error executing trade: %v", err)
	}

	return nil
}

func (uc *StartTradingBotUseCase) executeAnalysisAndTrade(tradingBot *entity.TradingBot) {
	currentBot, err := uc.tradingBotRepository.GetTradeByID(tradingBot.Id.GetValue())
	if err != nil || currentBot == nil || currentBot.GetStatus() != entity.StatusRunning {
		return
	}

	// Check if execution context wants to continue
	if !uc.executionContext.ShouldContinue() {
		return
	}

	// Use the public method to perform the analysis and trade
	if err := uc.ExecuteAnalysisAndTrade(tradingBot); err != nil {
		fmt.Printf("❌ Error in analysis and trade: %v\n", err)
	}
}

func (uc *StartTradingBotUseCase) getMarketData(symbol string) ([]vo.Kline, error) {
	binanceKlines, err := uc.client.NewKlinesService().
		Symbol(symbol).
		Interval("1h"). // TODO: Deixar dinamico.
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

func (uc *StartTradingBotUseCase) executeTradingDecision(tradingBot *entity.TradingBot, decision entity.TradingDecision) error {
	symbol := tradingBot.GetSymbol().GetValue()
	quantity := tradingBot.GetQuantity()

	switch decision {
	case entity.Buy:
		if tradingBot.GetIsPositioned() {
			return fmt.Errorf("this trading bot already has an open position")
		}
		fmt.Printf("🟢 Executing BUY order for %s, quantity: %.6f\n", symbol, quantity)

		// Get current price for entry price tracking
		klines, err := uc.getMarketData(symbol)
		if err != nil {
			fmt.Printf("❌ Error fetching current price for entry tracking: %v\n", err)
			return err
		}
		currentPrice := klines[len(klines)-1].Close()

		isOrderPlaced := uc.placeBuyOrder(symbol, quantity)
		if isOrderPlaced {
			// Set entry price when entering position
			tradingBot.SetEntryPrice(currentPrice)
			fmt.Printf("📈 Entry price set to: %.2f for bot %s\n", currentPrice, tradingBot.Id.GetValue())

			errPosition := tradingBot.GetIntoPosition()
			if errPosition != nil {
				return errPosition
			}
			errUpdate := uc.tradingBotRepository.Update(tradingBot)
			if errUpdate != nil {
				return errUpdate
			}
		}
		return nil
	case entity.Sell:
		if !tradingBot.GetIsPositioned() {
			return fmt.Errorf("this trading bot don't have an open position")
		}

		// Calculate and log the actual profit before selling
		klines, err := uc.getMarketData(symbol)
		if err != nil {
			fmt.Printf("❌ Error fetching current price for profit calculation: %v\n", err)
			return err
		}
		currentPrice := klines[len(klines)-1].Close()
		actualProfit := ((currentPrice - tradingBot.GetEntryPrice()) / tradingBot.GetEntryPrice()) * 100

		fmt.Printf("🔴 Executing SELL order for %s, quantity: %.6f (Profit: %.2f%%)\n", symbol, quantity, actualProfit)

		isOrderPlaced := uc.placeSellOrder(symbol, quantity)
		if isOrderPlaced {
			// Clear entry price when exiting position
			tradingBot.ClearEntryPrice()
			fmt.Printf("📉 Entry price cleared for bot %s after profitable sale\n", tradingBot.Id.GetValue())

			errPosition := tradingBot.GetOutOfPosition()
			if errPosition != nil {
				return errPosition
			}
			errUpdate := uc.tradingBotRepository.Update(tradingBot)
			if errUpdate != nil {
				return errUpdate
			}
		}
		return nil
	case entity.Hold:
		if tradingBot.GetIsPositioned() {
			// Log potential profit when holding a position
			klines, err := uc.getMarketData(symbol)
			if err == nil {
				currentPrice := klines[len(klines)-1].Close()
				potentialProfit := ((currentPrice - tradingBot.GetEntryPrice()) / tradingBot.GetEntryPrice()) * 100
				fmt.Printf("⏸ HOLDING position for %s (Current potential profit: %.2f%%)\n", symbol, potentialProfit)
			} else {
				fmt.Printf("⏸ HOLDING position for %s\n", symbol)
			}
		} else {
			fmt.Printf("⏸ HOLDING (no position) for %s\n", symbol)
		}
	}

	return nil
}

func (uc *StartTradingBotUseCase) placeBuyOrder(symbol string, quantity float64) bool {
	qtyStr := strconv.FormatFloat(quantity, 'f', 6, 64)

	order, err := uc.client.NewCreateOrderService().
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

func (uc *StartTradingBotUseCase) placeSellOrder(symbol string, quantity float64) bool {
	qtyStr := strconv.FormatFloat(quantity, 'f', 6, 64)

	order, err := uc.client.NewCreateOrderService().
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
