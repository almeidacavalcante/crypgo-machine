package usecase

import (
	"context"
	"crypgo-machine/src/application/service"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"fmt"
	"strconv"
	"time"
)

// BacktestTradingBotInput contains the parameters for running a backtest
type BacktestTradingBotInput struct {
	Symbol                 string                 `json:"symbol"`
	Strategy               string                 `json:"strategy"`
	StrategyParams         map[string]interface{} `json:"strategy_params"`
	StartDate              time.Time              `json:"start_date"`
	EndDate                time.Time              `json:"end_date"`
	InitialCapital         float64                `json:"initial_capital"`
	TradeAmount            float64                `json:"trade_amount"`
	TradingFees            float64                `json:"trading_fees"`
	MinimumProfitThreshold float64                `json:"minimum_profit_threshold"`
	Interval               string                 `json:"interval"`
	Currency               string                 `json:"currency"`
	Quantity               float64                `json:"quantity"`
	IntervalSeconds        int                    `json:"interval_seconds"`
}

// BacktestTradingBotUseCase performs backtesting using the same logic as live trading
type BacktestTradingBotUseCase struct {
	client external.BinanceClientInterface
}

// NewBacktestTradingBotUseCase creates a new BacktestTradingBotUseCase
func NewBacktestTradingBotUseCase(client external.BinanceClientInterface) *BacktestTradingBotUseCase {
	return &BacktestTradingBotUseCase{
		client: client,
	}
}

// Execute runs a backtest using historical data from Binance
func (uc *BacktestTradingBotUseCase) Execute(input BacktestTradingBotInput) (*service.BacktestResult, error) {
	// 1. Fetch historical data from Binance
	historicalData, err := uc.fetchHistoricalData(input.Symbol, input.StartDate, input.EndDate, input.Interval)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical data: %v", err)
	}

	if len(historicalData) == 0 {
		return nil, fmt.Errorf("no historical data available for the specified period")
	}

	fmt.Printf("📊 Loaded %d klines for backtesting %s from %s to %s\n",
		len(historicalData), input.Symbol,
		input.StartDate.Format("2006-01-02"), input.EndDate.Format("2006-01-02"))

	// 2. Create bot with specified strategy
	bot, err := uc.createBotForBacktest(input)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %v", err)
	}

	// 3. Set up backtest services
	dataSource := service.NewHistoricalMarketDataSource(historicalData, 100) // Same window as live
	executionContext := service.NewBacktestTradingExecutionContext(input.Symbol, input.InitialCapital)

	// 4. Create trading use case with backtest services
	tradingUseCase := NewStartTradingBotUseCaseWithServices(
		nil, // No repository needed for backtest
		nil, // No decision log repository needed for backtest
		uc.client,
		dataSource,
		executionContext,
	)

	// 5. Run the backtest simulation
	fmt.Printf("🚀 Starting backtest simulation...\n")

	processedCandles := 0
	totalCandles := len(historicalData)

	for dataSource.HasMoreData() {
		// Execute one analysis and trade decision
		if err := tradingUseCase.ExecuteAnalysisAndTrade(bot); err != nil {
			fmt.Printf("⚠️ Error during backtest at candle %d: %v\n", processedCandles, err)
		}

		// Advance to next candle
		if !dataSource.AdvanceToNext() {
			break
		}

		processedCandles++

		// Show progress every 10% of the way
		if processedCandles%max(1, totalCandles/10) == 0 {
			progress := float64(processedCandles) / float64(totalCandles) * 100
			fmt.Printf("📈 Progress: %.1f%% (%d/%d candles)\n", progress, processedCandles, totalCandles)
		}
	}

	// 6. Get and return results
	result := executionContext.GetResult()

	fmt.Printf("\n📈 BACKTEST SUMMARY:\n")
	fmt.Printf("   💰 Total P&L: %.2f BRL\n", result.TotalPnL)
	fmt.Printf("   📊 ROI: %.2f%%\n", result.ROI)
	fmt.Printf("   🎯 Win Rate: %.2f%%\n", result.WinRate)
	fmt.Printf("   🔄 Total Trades: %d\n", result.TotalTrades)
	fmt.Printf("   ✅ Winning: %d | ❌ Losing: %d\n", result.WinningTrades, result.LosingTrades)
	fmt.Printf("   📉 Max Drawdown: %.2f%%\n", result.MaxDrawdown)
	fmt.Printf("   💸 Trading Fees: %.2f BRL\n", result.TradingFees)

	return result, nil
}

// fetchHistoricalData retrieves historical klines from Binance for the specified period
func (uc *BacktestTradingBotUseCase) fetchHistoricalData(symbol string, startDate, endDate time.Time, interval string) ([]vo.Kline, error) {
	var allKlines []vo.Kline

	// Binance has a limit of 1000 klines per request, so we may need multiple requests
	currentStart := startDate

	for currentStart.Before(endDate) {
		// Calculate end time for this batch (max 1000 klines)
		var currentEnd time.Time
		switch interval {
		case "1h":
			currentEnd = currentStart.Add(1000 * time.Hour)
		case "5m":
			currentEnd = currentStart.Add(5 * time.Minute)
		case "4h":
			currentEnd = currentStart.Add(4000 * time.Hour)
		case "1d":
			currentEnd = currentStart.Add(1000 * 24 * time.Hour)
		default:
			currentEnd = currentStart.Add(1000 * time.Hour) // Default to 1h
		}
		
		if currentEnd.After(endDate) {
			currentEnd = endDate
		}

		fmt.Printf("📥 Fetching data from %s to %s...\n",
			currentStart.Format("2006-01-02 15:04"), currentEnd.Format("2006-01-02 15:04"))

		// Fetch klines for this period
		binanceKlines, err := uc.client.NewKlinesService().
			Symbol(symbol).
			Interval(interval).
			StartTime(currentStart.UnixMilli()).
			EndTime(currentEnd.UnixMilli()).
			Limit(1000).
			Do(context.Background())

		if err != nil {
			return nil, fmt.Errorf("error fetching klines: %v", err)
		}

		// Convert to domain klines
		for _, bkline := range binanceKlines {
			openPrice, _ := strconv.ParseFloat(bkline.Open, 64)
			closePrice, _ := strconv.ParseFloat(bkline.Close, 64)
			highPrice, _ := strconv.ParseFloat(bkline.High, 64)
			lowPrice, _ := strconv.ParseFloat(bkline.Low, 64)
			volumePrice, _ := strconv.ParseFloat(bkline.Volume, 64)

			kline, err := vo.NewKline(openPrice, closePrice, highPrice, lowPrice, volumePrice, bkline.CloseTime)
			if err != nil {
				return nil, fmt.Errorf("error creating kline: %v", err)
			}
			allKlines = append(allKlines, kline)
		}

		// Move to next batch
		currentStart = currentEnd.Add(time.Millisecond)
	}

	return allKlines, nil
}

// createBotForBacktest creates a trading bot configured for backtesting
func (uc *BacktestTradingBotUseCase) createBotForBacktest(input BacktestTradingBotInput) (*entity.TradingBot, error) {
	// Create symbol
	symbol, err := vo.NewSymbol(input.Symbol)
	if err != nil {
		return nil, fmt.Errorf("invalid symbol: %v", err)
	}

	// Use provided quantity or default
	quantity := input.Quantity
	if quantity <= 0 {
		quantity = 0.001 // Default quantity
	}

	// Create strategy with all parameters
	var strategy entity.TradingStrategy
	switch input.Strategy {
	case "MovingAverage":
		fast, ok1 := input.StrategyParams["FastWindow"].(float64)
		slow, ok2 := input.StrategyParams["SlowWindow"].(float64)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("invalid MovingAverage parameters")
		}

		// Check for minimum spread parameter
		if minimumSpread, ok := input.StrategyParams["MinimumSpread"].(float64); ok {
			spread, err := vo.NewMinimumSpread(minimumSpread)
			if err != nil {
				return nil, fmt.Errorf("invalid minimum spread: %v", err)
			}
			strategy = entity.NewMovingAverageStrategyWithSpread(int(fast), int(slow), spread)
		} else {
			strategy = entity.NewMovingAverageStrategy(int(fast), int(slow))
		}
	default:
		return nil, fmt.Errorf("unsupported strategy: %s", input.Strategy)
	}

	// Use provided currency or default
	currency := input.Currency
	if currency == "" {
		currency = "BRL"
	}

	// Use provided interval seconds or default
	intervalSeconds := input.IntervalSeconds
	if intervalSeconds <= 0 {
		intervalSeconds = 3600 // 1 hour default
	}

	// Create trading bot with all parameters
	bot := entity.NewTradingBot(
		symbol,
		quantity,
		strategy,
		intervalSeconds,
		input.InitialCapital,
		input.TradeAmount,
		currency,
		input.TradingFees,
		input.MinimumProfitThreshold,
		false,
	)

	return bot, nil
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
