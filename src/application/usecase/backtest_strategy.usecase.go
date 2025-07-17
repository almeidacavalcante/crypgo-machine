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
	StrategyName           string
	Symbol                 string
	Params                 map[string]interface{} // Strategy parameters (e.g., FastWindow, SlowWindow for MovingAverage)
	HistoricalData         []vo.Kline
	InitialCapital         float64
	TradeAmount            float64 // Fixed amount to use per trade (optional, if 0 uses all available capital)
	Currency               string
	StartDate              time.Time
	EndDate                time.Time
	TradingFees            float64 // Percentage fee per trade (e.g., 0.1 for 0.1%)
	MinimumProfitThreshold float64 // Minimum profit % required to sell (0 = sell at any profit)
}

type BacktestSimulator struct {
	strategy               entity.TradingStrategy
	result                 *entity.BacktestResult
	currentTrade           *entity.BacktestTrade
	currency               *vo.Currency
	tradingFees            float64
	tradeAmount            float64 // Fixed amount per trade, 0 means use all available capital
	minimumProfitThreshold float64 // Minimum profit % required to sell
	isPositioned           bool
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
		result:                 result,
		currency:               currency,
		tradingFees:            input.TradingFees,
		tradeAmount:            input.TradeAmount,
		minimumProfitThreshold: input.MinimumProfitThreshold,
		isPositioned:           false,
	}

	// Create strategy instance based on strategy name and parameters
	strategy, err := uc.createStrategy(input.StrategyName, input.Params)
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

func (uc *BacktestStrategyUseCase) createStrategy(strategyName string, params map[string]interface{}) (entity.TradingStrategy, error) {
	switch strategyName {
	case "MovingAverage":
		// Default parameters - conservative for reliable signals
		fastWindow := 7
		slowWindow := 40
		minimumSpreadValue := 0.1 // 0.1% spread requirement
		
		// Override with provided parameters if available
		if params != nil {
			if fw, ok := params["FastWindow"]; ok {
				if fwFloat, ok := fw.(float64); ok {
					fastWindow = int(fwFloat)
				}
			}
			if sw, ok := params["SlowWindow"]; ok {
				if swFloat, ok := sw.(float64); ok {
					slowWindow = int(swFloat)
				}
			}
			if ms, ok := params["MinimumSpread"]; ok {
				if msFloat, ok := ms.(float64); ok {
					minimumSpreadValue = msFloat
				}
			}
		}
		
		// Validate parameters
		if fastWindow <= 0 || slowWindow <= 0 || fastWindow >= slowWindow {
			return nil, fmt.Errorf("invalid MovingAverage parameters: FastWindow (%d) must be positive and less than SlowWindow (%d)", fastWindow, slowWindow)
		}
		
		if minimumSpreadValue < 0 {
			return nil, fmt.Errorf("invalid MinimumSpread parameter: %f (must be non-negative)", minimumSpreadValue)
		}
		
		minimumSpread, err := vo.NewMinimumSpread(minimumSpreadValue)
		if err != nil {
			return nil, fmt.Errorf("failed to create MinimumSpread: %w", err)
		}
		
		return entity.NewMovingAverageStrategyWithSpread(fastWindow, slowWindow, minimumSpread), nil
	default:
		return nil, fmt.Errorf("unsupported strategy: %s", strategyName)
	}
}

func (uc *BacktestStrategyUseCase) runSimulation(simulator *BacktestSimulator, historicalData []vo.Kline) error {
	// Create a dummy trading bot for strategy decisions - SHARED across all iterations
	symbol, _ := vo.NewSymbol("SOLBRL") // This will be overridden by the actual symbol
	dummyBot := entity.NewTradingBot(symbol, 1.0, simulator.strategy, 60, 10000.0, 1000.0, "BRL", 0.001, 0.0)

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

		// Get all historical data from beginning up to current point
		windowData := historicalData[0 : i+1] // All data from start to current

		// Get current price and time first
		currentTime := time.Unix(kline.CloseTime()/1000, 0)
		currentPrice := kline.Close()

		// CRITICAL FIX: Ensure both states are synchronized BEFORE strategy decision
		uc.syncPositionStates(simulator, dummyBot)

		// Get strategy decision - CRITICAL FIX: Pass reference, not copy
		analysisResult := simulator.strategy.Decide(windowData, dummyBot)
		
		// DEBUG: Log critical state for sell decisions
		if analysisResult.Decision == entity.Sell && simulator.isPositioned {
			fmt.Printf("🔍 SELL DECISION DEBUG | Bot Entry: R$%.2f | Sim Entry: R$%.2f | Current: R$%.2f | Profit: %.2f%%\n",
				dummyBot.GetEntryPrice(), 
				simulator.currentTrade.GetEntryPrice(), 
				currentPrice,
				analysisResult.AnalysisData["possibleProfit"].(float64))
		}

		// Calculate potential profit if positioned
		var potentialProfit float64 = 0.0
		if simulator.isPositioned && dummyBot.GetEntryPrice() > 0 {
			potentialProfit = ((currentPrice - dummyBot.GetEntryPrice()) / dummyBot.GetEntryPrice()) * 100
		}

		// Enhanced DEBUG: Log detailed state every 50 iterations
		if i%50 == 0 {
			if simulator.isPositioned {
				fmt.Printf("📊 [%d] %s | Price: R$%.2f | Entry: R$%.2f | Profit: %.2f%% | Decision: %s\n",
					i, currentTime.Format("2006-01-02 15:04"), currentPrice, dummyBot.GetEntryPrice(), potentialProfit, analysisResult.Decision)
			} else {
				fmt.Printf("📊 [%d] %s | Price: R$%.2f | No Position | Decision: %s\n",
					i, currentTime.Format("2006-01-02 15:04"), currentPrice, analysisResult.Decision)
			}
		}

		// Execute trading decision
		err := uc.executeDecision(simulator, analysisResult.Decision, currentPrice, currentTime, analysisResult.AnalysisData)
		if err != nil {
			return fmt.Errorf("failed to execute decision at %s: %w", currentTime, err)
		}
	}

	// Check final position - apply profit protection (never close at loss)
	if simulator.currentTrade != nil && simulator.currentTrade.IsOpen() {
		lastKline := historicalData[len(historicalData)-1]
		lastTime := time.Unix(lastKline.CloseTime()/1000, 0)
		lastPrice := lastKline.Close()

		// Calculate profit before potential close
		entryPrice := simulator.currentTrade.GetEntryPrice()
		potentialProfit := ((lastPrice - entryPrice) / entryPrice) * 100

		if potentialProfit > 0 && potentialProfit >= simulator.minimumProfitThreshold {
			// Only close if profitable AND meets minimum threshold
			err := simulator.currentTrade.Close(lastPrice, lastTime, simulator.currency)
			if err != nil {
				return fmt.Errorf("failed to close final trade: %w", err)
			}

			finalProfit := simulator.currentTrade.GetProfitLoss()
			fmt.Printf("🏁 FINAL CLOSE (TARGET REACHED) %s | Price: R$%.2f | Entry: R$%.2f | Profit: %.2f%% | Target: %.2f%% | P&L: %s\n",
				simulator.result.GetSymbol().GetValue(), lastPrice, entryPrice, potentialProfit, simulator.minimumProfitThreshold, finalProfit.String())

			err = simulator.result.AddTrade(simulator.currentTrade)
			if err != nil {
				return fmt.Errorf("failed to add final trade: %w", err)
			}
		} else if potentialProfit > 0 {
			// Profitable but below threshold
			fmt.Printf("🎯 HOLDING FOR TARGET %s | Price: R$%.2f | Entry: R$%.2f | Profit: %.2f%% | Target: %.2f%% | WAITING FOR TARGET\n",
				simulator.result.GetSymbol().GetValue(), lastPrice, entryPrice, potentialProfit, simulator.minimumProfitThreshold)
			fmt.Printf("📊 Position remains open - target not reached yet\n")
		} else {
			// Keep position open - never sell at loss
			fmt.Printf("💎 HOLDING POSITION %s | Price: R$%.2f | Entry: R$%.2f | Loss: %.2f%% | NEVER SELL AT LOSS\n",
				simulator.result.GetSymbol().GetValue(), lastPrice, entryPrice, potentialProfit)
			fmt.Printf("📊 Position remains open with potential to recover\n")
		}
	}

	// Print backtest summary
	totalTrades := simulator.result.GetTotalTrades()
	winRate := simulator.result.GetWinRate()
	finalROI := simulator.result.GetROI()
	totalPL := simulator.result.GetTotalProfitLoss()

	fmt.Printf("\n📈 BACKTEST SUMMARY:\n")
	fmt.Printf("   💰 Total P&L: %s\n", totalPL.String())
	fmt.Printf("   📊 ROI: %.2f%%\n", finalROI)
	fmt.Printf("   🎯 Win Rate: %s\n", winRate.String())
	fmt.Printf("   🔄 Total Trades: %d\n", totalTrades)
	fmt.Printf("   ✅ Winning: %d | ❌ Losing: %d\n", simulator.result.GetWinningTrades(), simulator.result.GetLosingTrades())

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
			// Open buy position
			reason := "unknown"
			if r, ok := analysisData["reason"].(string); ok {
				reason = r
			}

			// Calculate quantity based on trade amount or available capital
			var amountToUse float64
			var logMessage string

			if simulator.tradeAmount > 0 {
				// Use fixed trade amount
				availableCapital := simulator.result.GetFinalCapital()
				if simulator.tradeAmount > availableCapital {
					// Not enough capital for fixed amount, skip this trade
					fmt.Printf("⚠️ SKIP BUY %s | Price: R$%.2f | Insufficient capital: R$%.2f < R$%.2f | %s at %s\n",
						symbol.GetValue(), price, availableCapital, simulator.tradeAmount, reason, timestamp.Format("2006-01-02 15:04"))
					return nil
				}
				amountToUse = simulator.tradeAmount
				logMessage = fmt.Sprintf("🟢 BUY %s | Price: R$%.2f | Fixed Amount: R$%.2f",
					symbol.GetValue(), price, amountToUse)
			} else {
				// Use all available capital
				amountToUse = simulator.result.GetFinalCapital()
				logMessage = fmt.Sprintf("🟢 BUY %s | Price: R$%.2f | All Capital: R$%.2f",
					symbol.GetValue(), price, amountToUse)
			}

			feeAdjustedPrice := price * (1 + simulator.tradingFees/100)
			quantity := amountToUse / feeAdjustedPrice

			// Enhanced log with trading info
			fmt.Printf("%s | Qty: %.6f | %s at %s\n",
				logMessage, quantity, reason, timestamp.Format("2006-01-02 15:04"))

			trade := entity.NewBacktestTrade(symbol, entity.Buy, price, quantity, timestamp, reason)
			simulator.currentTrade = trade
			simulator.isPositioned = true
		}

	case entity.Sell:
		if simulator.isPositioned && simulator.currentTrade != nil {
			// Calculate profit before closing
			entryPrice := simulator.currentTrade.GetEntryPrice()
			rawProfit := ((price - entryPrice) / entryPrice) * 100

			// CRITICAL PROTECTION: Double-check profit before allowing sale
			if rawProfit < 0 {
				fmt.Printf("🚨 BLOCKED SALE AT LOSS | Price: R$%.2f | Entry: R$%.2f | Loss: %.2f%% | REASON: %s | at %s\n",
					price, entryPrice, rawProfit, analysisData["reason"], timestamp.Format("2006-01-02 15:04"))
				return nil // Block the sale
			}
			
			// NEW PROTECTION: Check minimum profit threshold
			if rawProfit < simulator.minimumProfitThreshold {
				fmt.Printf("🎯 WAITING FOR TARGET | Price: R$%.2f | Entry: R$%.2f | Profit: %.2f%% | Target: %.2f%% | at %s\n",
					price, entryPrice, rawProfit, simulator.minimumProfitThreshold, timestamp.Format("2006-01-02 15:04"))
				return nil // Block the sale until target is reached
			}

			// Close position
			feeAdjustedPrice := price * (1 - simulator.tradingFees/100)
			err := simulator.currentTrade.Close(feeAdjustedPrice, timestamp, simulator.currency)
			if err != nil {
				return err
			}

			// Calculate final profit after fees
			finalProfit := simulator.currentTrade.GetProfitLoss()

			// Enhanced log with profit info
			fmt.Printf("🔴 SELL %s | Price: R$%.2f | Entry: R$%.2f | Raw Profit: %.2f%% | Final P&L: %s | at %s\n",
				symbol.GetValue(), price, entryPrice, rawProfit, finalProfit.String(), timestamp.Format("2006-01-02 15:04"))

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

// syncPositionStates ensures simulator and bot have same position state and entry price
func (uc *BacktestStrategyUseCase) syncPositionStates(simulator *BacktestSimulator, bot *entity.TradingBot) {
	simPos := simulator.isPositioned
	botPos := bot.GetIsPositioned()

	// Only try to change state if they're different
	if simPos && !botPos {
		err := bot.GetIntoPosition() // Bot not positioned, simulator is - put bot into position
		if err != nil {
			fmt.Printf("ERROR: GetIntoPosition failed: %v\n", err)
		}
		// Sync entry price from current trade
		if simulator.currentTrade != nil {
			bot.SetEntryPrice(simulator.currentTrade.GetEntryPrice())
		}
	} else if !simPos && botPos {
		err := bot.GetOutOfPosition() // Bot positioned, simulator not - take bot out of position
		if err != nil {
			fmt.Printf("ERROR: GetOutOfPosition failed: %v\n", err)
		}
		// Clear entry price when not positioned
		bot.ClearEntryPrice()
	} else if simPos && botPos {
		// Both positioned - ensure entry price is synced
		if simulator.currentTrade != nil {
			bot.SetEntryPrice(simulator.currentTrade.GetEntryPrice())
		}
	}
}

