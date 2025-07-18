package main

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/application/usecase"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/infra/api"
	"crypgo-machine/src/infra/database"
	"crypgo-machine/src/infra/external"
	"crypgo-machine/src/infra/notification"
	"crypgo-machine/src/infra/queue"
	infraRepository "crypgo-machine/src/infra/repository"
	"fmt"
	"github.com/adshao/go-binance/v2"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
)

func main() {
	fmt.Println("🚀 Starting Binance Trading Bot...")
	loadEnv()

	client := binance.NewClient(
		os.Getenv("BINANCE_API_KEY"),
		os.Getenv("BINANCE_SECRET_KEY"),
	)

	dbConnection, err := database.NewDatabaseConnection(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	if err != nil {
		log.Fatalf("❌ Error connecting to database: %v", err)
	}
	fmt.Println("✅ Database connection established.")

	tradingBotRepository := infraRepository.NewTradingBotRepositoryDatabase(dbConnection.DB)
	rabbit, err := queue.NewRabbitQMAdapter(os.Getenv("RABBIT_MQ_URL"))
	if err != nil {
		log.Fatal(err)
	}
	decisionLogRepository := infraRepository.NewTradingDecisionLogRepositoryDatabase(dbConnection.DB)

	// Email service setup
	emailService := notification.NewEmailService()
	targetEmail := os.Getenv("TARGET_EMAIL")
	if targetEmail == "" {
		targetEmail = "jalmeidacn@gmail.com" // fallback
	}

	notificationConsumer := notification.NewEmailNotificationConsumer(rabbit, "trading_bot", "email.notification.queue", emailService, targetEmail)
	go func() {
		err := notificationConsumer.Start()
		if err != nil {
			log.Fatalf("❌ Error starting email notification consumer: %v", err)
		} else {
			fmt.Println("✅ Email notification consumer started successfully.")
		}
	}()

	createTradingBotUseCase := usecase.NewCreateTradingBotUseCase(tradingBotRepository, *client, rabbit, "trading_bot")
	createTradingBotController := api.NewCreateTradingBotController(createTradingBotUseCase)
	http.HandleFunc("/api/v1/trading/create_trading_bot", createTradingBotController.Handle)

	listAllTradingBotsUseCase := usecase.NewListAllTradingBotsUseCase(tradingBotRepository)
	listAllTradingBotsController := api.NewListAllTradingBotsController(listAllTradingBotsUseCase)
	http.HandleFunc("/api/v1/trading/list", listAllTradingBotsController.Handle)

	binanceWrapper := external.NewBinanceClientWrapper(client)
	startTradingBotUseCase := usecase.NewStartTradingBotUseCaseWithMessaging(tradingBotRepository, decisionLogRepository, binanceWrapper, rabbit, "trading_bot")
	startTradingBotController := api.NewStartTradingBotController(startTradingBotUseCase)
	http.HandleFunc("/api/v1/trading/start", startTradingBotController.Handle)

	// Auto-recovery: restart all running bots after server restart
	fmt.Println("🔧 About to start auto-recovery...")
	if err := recoverRunningBots(tradingBotRepository, startTradingBotUseCase); err != nil {
		fmt.Printf("⚠️ Auto-recovery completed with some errors: %v\n", err)
	} else {
		fmt.Println("🔧 Auto-recovery finished successfully")
	}

	stopTradingBotUseCase := usecase.NewStopTradingBotUseCase(tradingBotRepository)
	stopTradingBotController := api.NewStopTradingBotController(stopTradingBotUseCase)
	http.HandleFunc("/api/v1/trading/stop", stopTradingBotController.Handle)

	backtestStrategyUseCase := usecase.NewBacktestStrategyUseCase()
	historicalDataService := external.NewBinanceHistoricalDataService(binanceWrapper)
	backtestStrategyController := api.NewBacktestStrategyController(backtestStrategyUseCase, historicalDataService)
	http.HandleFunc("/api/v1/trading/backtest", backtestStrategyController.Handle)

	// Serve static files for dashboard
	http.Handle("/", http.FileServer(http.Dir("./web/")))

	fmt.Println("🚀 Listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("❌ Error starting server: %v", err)
	}

	defer dbConnection.DB.Close()
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("❌ Error loading .env file: %v", err)
	}
}

// recoverRunningBots finds all trading bots with RUNNING status and restarts their trading loops
func recoverRunningBots(tradingBotRepository repository.TradingBotRepository, startTradingBotUseCase *usecase.StartTradingBotUseCase) error {
	fmt.Println("🔄 Starting auto-recovery process...")

	// Get all bots with RUNNING status
	runningBots, err := tradingBotRepository.GetTradingBotsByStatus(entity.StatusRunning)
	if err != nil {
		fmt.Printf("❌ Error querying running bots: %v\n", err)
		return err
	}

	if len(runningBots) == 0 {
		fmt.Println("ℹ️ No running trading bots found to recover")
		return nil
	}

	fmt.Printf("🔍 Found %d running trading bot(s) to recover\n", len(runningBots))

	// Track recovery results
	successCount := 0
	errorCount := 0

	// Restart each running bot
	for _, bot := range runningBots {
		botId := bot.Id.GetValue()
		symbol := bot.GetSymbol().GetValue()
		
		fmt.Printf("⚡ Recovering bot %s (%s)...\n", botId, symbol)
		
		// For auto-recovery, we need to reset the bot status to STOPPED first
		// because the server restart killed the actual trading loops but left the status as RUNNING
		if err := bot.Stop(); err != nil {
			fmt.Printf("❌ Failed to reset bot status for recovery %s (%s): %v\n", botId, symbol, err)
			errorCount++
			continue
		}
		
		// Update the bot status in database
		if err := tradingBotRepository.Update(bot); err != nil {
			fmt.Printf("❌ Failed to update bot status for recovery %s (%s): %v\n", botId, symbol, err)
			errorCount++
			continue
		}
		
		// Now use existing StartTradingBotUseCase to restart the bot
		input := usecase.InputStartTradingBot{
			TradingBotId: botId,
		}
		
		if err := startTradingBotUseCase.Execute(input); err != nil {
			fmt.Printf("❌ Failed to recover bot %s (%s): %v\n", botId, symbol, err)
			errorCount++
		} else {
			fmt.Printf("✅ Successfully recovered bot %s (%s)\n", botId, symbol)
			successCount++
		}
	}

	// Summary
	fmt.Printf("📊 Auto-recovery completed: %d successful, %d failed\n", successCount, errorCount)
	
	if errorCount > 0 {
		return fmt.Errorf("auto-recovery completed with %d errors", errorCount)
	}
	
	return nil
}
