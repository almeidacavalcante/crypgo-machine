package main

import (
	"crypgo-machine/src/application/usecase"
	"crypgo-machine/src/infra/external"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/adshao/go-binance/v2"
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		// Don't fail if .env doesn't exist, just continue
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Command line flags
	var (
		symbol                 = flag.String("symbol", "SOLBRL", "Trading symbol (e.g., SOLBRL, BTCBRL)")
		strategy               = flag.String("strategy", "MovingAverage", "Strategy name")
		fastWindow             = flag.Int("fast", 5, "Fast moving average window")
		slowWindow             = flag.Int("slow", 10, "Slow moving average window")
		startDateStr           = flag.String("start", "", "Start date (YYYY-MM-DD) - required")
		endDateStr             = flag.String("end", "", "End date (YYYY-MM-DD) - required")
		initialCapital         = flag.Float64("capital", 10000.0, "Initial capital in BRL")
		tradeAmount            = flag.Float64("amount", 5000.0, "Trade amount per operation in BRL")
		tradingFees            = flag.Float64("fees", 0.01, "Trading fees percentage (0.01 = 0.01%)")
		minimumProfitThreshold = flag.Float64("min-profit", 2.0, "Minimum profit threshold percentage")
		interval               = flag.String("interval", "30m", "Kline interval (1m, 5m, 15m, 30m, 1h, 4h, 1d)")
		currency               = flag.String("currency", "BRL", "Currency for calculations")
		quantity               = flag.Float64("quantity", 0.001, "Quantity per trade (for crypto pairs)")
		intervalSeconds        = flag.Int("interval-seconds", 1800, "Interval in seconds for bot operations")
		minimumSpread          = flag.Float64("min-spread", 0.1, "Minimum spread percentage for anti-whipsaw")
		outputFile             = flag.String("output", "", "Output file for results (optional)")
		apiKey                 = flag.String("api-key", "", "Binance API key (or use BINANCE_API_KEY env var)")
		secretKey              = flag.String("secret-key", "", "Binance secret key (or use BINANCE_SECRET_KEY env var)")
		verbose                = flag.Bool("verbose", false, "Enable verbose output")
		quiet                  = flag.Bool("quiet", false, "Suppress progress output")
	)
	flag.Parse()

	// Validate required parameters
	if *startDateStr == "" || *endDateStr == "" {
		fmt.Println("❌ Error: start and end dates are required")
		fmt.Println("\nUsage examples:")
		fmt.Println("  # Basic backtest with SOLBRL")
		fmt.Println("  go run cmd/backtest/main.go -start=2024-01-01 -end=2024-01-31")
		fmt.Println("\n  # Advanced backtest with custom parameters")
		fmt.Println("  go run cmd/backtest/main.go \\")
		fmt.Println("    -start=2024-01-01 -end=2024-01-31 \\")
		fmt.Println("    -symbol=BTCBRL -fast=5 -slow=10 \\")
		fmt.Println("    -capital=10000 -amount=5000 \\")
		fmt.Println("    -min-profit=2.5 -min-spread=0.7 \\")
		fmt.Println("    -interval=1h -output=results.json")
		fmt.Println("\nParameters:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", *startDateStr)
	if err != nil {
		log.Fatalf("❌ Error parsing start date: %v", err)
	}

	endDate, err := time.Parse("2006-01-02", *endDateStr)
	if err != nil {
		log.Fatalf("❌ Error parsing end date: %v", err)
	}

	if endDate.Before(startDate) {
		log.Fatal("❌ Error: end date must be after start date")
	}

	// Get API credentials
	binanceAPIKey := *apiKey
	binanceSecretKey := *secretKey

	if binanceAPIKey == "" {
		binanceAPIKey = os.Getenv("BINANCE_API_KEY")
	}
	if binanceSecretKey == "" {
		binanceSecretKey = os.Getenv("BINANCE_SECRET_KEY")
	}

	if binanceAPIKey == "" || binanceSecretKey == "" {
		log.Fatal("❌ Error: Binance API credentials are required. Set BINANCE_API_KEY and BINANCE_SECRET_KEY environment variables or use -api-key and -secret-key flags")
	}

	// Create Binance client
	binanceClient := binance.NewClient(binanceAPIKey, binanceSecretKey)
	client := external.NewBinanceClientWrapper(binanceClient)

	// Create backtest use case
	useCase := usecase.NewBacktestTradingBotUseCase(client)

	// Prepare input with all dynamic parameters
	input := usecase.BacktestTradingBotInput{
		Symbol:   *symbol,
		Strategy: *strategy,
		StrategyParams: map[string]interface{}{
			"FastWindow":    float64(*fastWindow),
			"SlowWindow":    float64(*slowWindow),
			"MinimumSpread": *minimumSpread,
		},
		StartDate:              startDate,
		EndDate:                endDate,
		InitialCapital:         *initialCapital,
		TradeAmount:            *tradeAmount,
		TradingFees:            *tradingFees,
		MinimumProfitThreshold: *minimumProfitThreshold,
		Interval:               *interval,
		Currency:               *currency,
		Quantity:               *quantity,
		IntervalSeconds:        *intervalSeconds,
	}

	// Print configuration unless quiet mode
	if !*quiet {
		fmt.Printf("🚀 Starting backtest with configuration:\n")
		fmt.Printf("   Symbol: %s\n", *symbol)
		fmt.Printf("   Strategy: %s (Fast: %d, Slow: %d)\n", *strategy, *fastWindow, *slowWindow)
		fmt.Printf("   Period: %s to %s\n", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
		fmt.Printf("   Initial Capital: %.2f %s\n", *initialCapital, *currency)
		fmt.Printf("   Trade Amount: %.2f %s\n", *tradeAmount, *currency)
		fmt.Printf("   Quantity per Trade: %.6f\n", *quantity)
		fmt.Printf("   Trading Fees: %.3f%%\n", *tradingFees)
		fmt.Printf("   Minimum Profit Threshold: %.2f%%\n", *minimumProfitThreshold)
		fmt.Printf("   Minimum Spread: %.2f%%\n", *minimumSpread)
		fmt.Printf("   Interval: %s (%d seconds)\n", *interval, *intervalSeconds)
		if *verbose {
			fmt.Printf("   Currency: %s\n", *currency)
			fmt.Printf("   API Key: %s...\n", binanceAPIKey[:minInt(len(binanceAPIKey), 8)])
		}
		fmt.Printf("\n")
	}

	// Execute backtest
	result, err := useCase.Execute(input)
	if err != nil {
		log.Fatalf("❌ Backtest failed: %v", err)
	}

	// Save results to file if specified
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			log.Printf("⚠️ Warning: Failed to create output file: %v", err)
		} else {
			defer file.Close()

			encoder := json.NewEncoder(file)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(result); err != nil {
				log.Printf("⚠️ Warning: Failed to write results to file: %v", err)
			} else {
				fmt.Printf("💾 Results saved to: %s\n", *outputFile)
			}
		}
	}

	// Print detailed trade analysis
	if len(result.Trades) > 0 {
		fmt.Printf("\n📊 DETAILED TRADE ANALYSIS:\n")
		fmt.Printf("   Trade #  | Entry Price | Exit Price |    P&L    |   P&L%%   | Duration\n")
		fmt.Printf("   ---------|-------------|------------|-----------|---------|----------\n")

		for i, trade := range result.Trades {
			duration := trade.ExitTime.Sub(trade.EntryTime)
			fmt.Printf("   %8d | %11.2f | %10.2f | %9.2f | %7.2f%% | %8s\n",
				i+1, trade.EntryPrice, trade.ExitPrice, trade.PnL, trade.PnLPercentage,
				duration.Truncate(time.Hour).String())
		}
	}

	fmt.Printf("\n✅ Backtest completed successfully!\n")
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
