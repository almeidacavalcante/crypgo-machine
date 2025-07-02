package usecase

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"fmt"
	"time"
)

type BacktestStrategyUseCase struct {
	// Dependencies can be added here if needed (e.g., for fetching historical data)
}

func NewBacktestStrategyUseCase() *BacktestStrategyUseCase {
	return &BacktestStrategyUseCase{}
}

type InputBacktestStrategy struct {
	StrategyName   string
	Symbol         string
	HistoricalData []vo.Kline
	InitialCapital float64
	Currency       string
	StartDate      time.Time
	EndDate        time.Time
	TradingFees    float64 // Percentage fee per trade (e.g., 0.1 for 0.1%)
}

type BacktestSimulator struct {
	strategy     entity.TradingStrategy
	result       *entity.BacktestResult
	currentTrade *entity.BacktestTrade
	currency     *vo.Currency
	tradingFees  float64
	isPositioned bool
}

func (uc *BacktestStrategyUseCase) Execute(input InputBacktestStrategy) (*entity.BacktestResult, error) {
	// Validate input
	if err := uc.validateInput(input); err != nil {
		return nil, err
	}

	// Create value objects
	symbol, err := vo.NewSymbol(input.Symbol)
	if err != nil {
		return nil, fmt.Errorf("invalid symbol: %w", err)
	}

	currency, err := vo.NewCurrency(input.Currency)
	if err != nil {
		return nil, fmt.Errorf("invalid currency: %w", err)
	}

	// Create backtest result
	result, err := entity.NewBacktestResult(
		input.StrategyName,
		symbol,
		input.StartDate,
		input.EndDate,
		input.InitialCapital,
		*currency,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create backtest result: %w", err)
	}

	// Create simulator
	simulator := &BacktestSimulator{
		result:       result,
		currency:     currency,
		tradingFees:  input.TradingFees,
		isPositioned: false,
	}

	// Create strategy instance based on strategy name
	strategy, err := uc.createStrategy(input.StrategyName)
	if err != nil {
		return nil, fmt.Errorf("failed to create strategy: %w", err)
	}
	simulator.strategy = strategy

	// Run simulation
	err = uc.runSimulation(simulator, input.HistoricalData)
	if err != nil {
		return nil, fmt.Errorf("simulation failed: %w", err)
	}

	return result, nil
}

func (uc *BacktestStrategyUseCase) validateInput(input InputBacktestStrategy) error {
	if input.StrategyName == "" {
		return fmt.Errorf("strategy name is required")
	}

	if input.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if len(input.HistoricalData) == 0 {
		return fmt.Errorf("historical data is required")
	}

	if input.InitialCapital <= 0 {
		return fmt.Errorf("initial capital must be positive")
	}

	if input.Currency == "" {
		return fmt.Errorf("currency is required")
	}

	if input.StartDate.After(input.EndDate) {
		return fmt.Errorf("start date must be before end date")
	}

	if input.TradingFees < 0 {
		return fmt.Errorf("trading fees cannot be negative")
	}

	return nil
}

func (uc *BacktestStrategyUseCase) createStrategy(strategyName string) (entity.TradingStrategy, error) {
	switch strategyName {
	case "MovingAverage":
		// Conservative parameters for reliable signals
		minimumSpread, _ := vo.NewMinimumSpread(0.5) // 0.5% spread requirement
		return entity.NewMovingAverageStrategyWithSpread(7, 40, minimumSpread), nil
	default:
		return nil, fmt.Errorf("unsupported strategy: %s", strategyName)
	}
}

func (uc *BacktestStrategyUseCase) runSimulation(simulator *BacktestSimulator, historicalData []vo.Kline) error {
	// Create a dummy trading bot for strategy decisions - SHARED across all iterations
	symbol, _ := vo.NewSymbol("SOLBRL") // This will be overridden by the actual symbol
	dummyBot := entity.NewTradingBot(symbol, 1.0, simulator.strategy, 60)

	// CRITICAL FIX: Start the bot so it can properly track position state
	err := dummyBot.Start()
	if err != nil {
		return fmt.Errorf("failed to start dummy bot: %w", err)
	}

	// Process each data point
	for i, kline := range historicalData {
		// Ensure we have enough data for the strategy
		if i < 21 { // Minimum required for most strategies
			continue
		}

		// Get the relevant window of data for this point
		windowData := historicalData[max(0, i-99) : i+1] // Last 100 periods including current

		// CRITICAL FIX: Ensure both states are synchronized BEFORE strategy decision
		uc.syncPositionStates(simulator, dummyBot)

		// DEBUG: Log state before decision
		if i%50 == 0 {
			fmt.Printf("DEBUG: i=%d, sim.positioned=%v, bot.positioned=%v\n", 
				i, simulator.isPositioned, dummyBot.GetIsPositioned())
		}

		// Get strategy decision - CRITICAL FIX: Pass reference, not copy
		analysisResult := simulator.strategy.Decide(windowData, dummyBot)

		currentTime := time.Unix(kline.CloseTime()/1000, 0)
		currentPrice := kline.Close()

		// Execute trading decision
		err := uc.executeDecision(simulator, analysisResult.Decision, currentPrice, currentTime, analysisResult.AnalysisData)
		if err != nil {
			return fmt.Errorf("failed to execute decision at %s: %w", currentTime, err)
		}
	}

	// Close any open position at the end
	if simulator.currentTrade != nil && simulator.currentTrade.IsOpen() {
		lastKline := historicalData[len(historicalData)-1]
		lastTime := time.Unix(lastKline.CloseTime()/1000, 0)
		lastPrice := lastKline.Close()

		err := simulator.currentTrade.Close(lastPrice, lastTime, simulator.currency)
		if err != nil {
			return fmt.Errorf("failed to close final trade: %w", err)
		}

		fmt.Printf("DEBUG: Adding trade to result (final cleanup)\n")
		err = simulator.result.AddTrade(simulator.currentTrade)
		if err != nil {
			return fmt.Errorf("failed to add final trade: %w", err)
		}
	}

	return nil
}

func (uc *BacktestStrategyUseCase) executeDecision(
	simulator *BacktestSimulator,
	decision entity.TradingDecision,
	price float64,
	timestamp time.Time,
	analysisData map[string]interface{},
) error {
	symbol := simulator.result.GetSymbol()

	switch decision {
	case entity.Buy:
		if !simulator.isPositioned {
			// log
			fmt.Printf("Opening position for %s at price %.2f at %s\n", symbol.GetValue(), price, timestamp)
			// Open buy position
			reason := "unknown"
			if r, ok := analysisData["reason"].(string); ok {
				reason = r
			}

			// Calculate quantity based on available capital
			availableCapital := simulator.result.GetFinalCapital()
			feeAdjustedPrice := price * (1 + simulator.tradingFees/100)
			quantity := availableCapital / feeAdjustedPrice

			trade := entity.NewBacktestTrade(symbol, entity.Buy, price, quantity, timestamp, reason)
			simulator.currentTrade = trade
			simulator.isPositioned = true
		}

	case entity.Sell:
		if simulator.isPositioned && simulator.currentTrade != nil {
			// log
			fmt.Printf("Closing position for %s at price %.2f at %s\n", symbol.GetValue(), price, timestamp)
			// Close position
			feeAdjustedPrice := price * (1 - simulator.tradingFees/100)
			err := simulator.currentTrade.Close(feeAdjustedPrice, timestamp, simulator.currency)
			if err != nil {
				return err
			}

			fmt.Printf("DEBUG: Adding trade to result (SELL case)\n")
			err = simulator.result.AddTrade(simulator.currentTrade)
			if err != nil {
				return err
			}

			simulator.currentTrade = nil
			simulator.isPositioned = false
		}

	case entity.Hold:
		// nothing
	}

	return nil
}

// syncPositionStates ensures simulator and bot have same position state
func (uc *BacktestStrategyUseCase) syncPositionStates(simulator *BacktestSimulator, bot *entity.TradingBot) {
	simPos := simulator.isPositioned
	botPos := bot.GetIsPositioned()
	
	// Only try to change state if they're different
	if simPos && !botPos {
		err := bot.GetIntoPosition() // Bot not positioned, simulator is - put bot into position
		if err != nil {
			fmt.Printf("ERROR: GetIntoPosition failed: %v\n", err)
		}
	} else if !simPos && botPos {
		err := bot.GetOutOfPosition() // Bot positioned, simulator not - take bot out of position
		if err != nil {
			fmt.Printf("ERROR: GetOutOfPosition failed: %v\n", err)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
