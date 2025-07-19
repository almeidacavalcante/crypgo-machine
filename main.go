package main

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/application/usecase"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/infra/api"
	"crypgo-machine/src/infra/auth"
	"crypgo-machine/src/infra/database"
	"crypgo-machine/src/infra/external"
	"crypgo-machine/src/infra/http/controller"
	"crypgo-machine/src/infra/middleware"
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

	// Telegram service setup
	telegramService := notification.NewTelegramService()
	telegramConsumer := notification.NewTelegramNotificationConsumer(rabbit, "trading_bot", "telegram.notification.queue", telegramService)
	go func() {
		err := telegramConsumer.Start()
		if err != nil {
			log.Printf("❌ Error starting Telegram notification consumer: %v", err)
		} else {
			fmt.Println("✅ Telegram notification consumer started successfully.")
		}
	}()

	// Authentication setup
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "crypgo-super-secret-key-2024-production" // fallback - change in production
	}
	
	validEmail := "jalmeidacn@gmail.com"
	validPassword := "CrypGo2024#StrongPass!" // Strong password
	
	jwtService := auth.NewJWTService(jwtSecret, "crypgo-machine")
	authUseCase := usecase.NewAuthUseCase(jwtService, validEmail, validPassword)
	authController := api.NewAuthController(authUseCase)
	authMiddleware := middleware.NewAuthMiddleware(authUseCase)

	// Public routes
	healthController := api.NewHealthController()
	http.HandleFunc("/api/v1/health", healthController.Health)
	
	// Public auth routes
	http.HandleFunc("/api/v1/auth/login", authController.Login)
	http.HandleFunc("/api/v1/auth/refresh", authController.RefreshToken)
	http.HandleFunc("/api/v1/auth/validate", authController.ValidateToken)

	// Protected trading routes
	createTradingBotUseCase := usecase.NewCreateTradingBotUseCase(tradingBotRepository, *client, rabbit, "trading_bot")
	createTradingBotController := api.NewCreateTradingBotController(createTradingBotUseCase)
	http.HandleFunc("/api/v1/trading/create_trading_bot", authMiddleware.RequireAuth(createTradingBotController.Handle))

	listAllTradingBotsUseCase := usecase.NewListAllTradingBotsUseCase(tradingBotRepository)
	listAllTradingBotsController := api.NewListAllTradingBotsController(listAllTradingBotsUseCase)
	http.HandleFunc("/api/v1/trading/list", authMiddleware.RequireAuth(listAllTradingBotsController.Handle))

	binanceWrapper := external.NewBinanceClientWrapper(client)
	startTradingBotUseCase := usecase.NewStartTradingBotUseCaseWithMessaging(tradingBotRepository, decisionLogRepository, binanceWrapper, rabbit, "trading_bot")
	startTradingBotController := api.NewStartTradingBotController(startTradingBotUseCase)
	http.HandleFunc("/api/v1/trading/start", authMiddleware.RequireAuth(startTradingBotController.Handle))

	// Auto-recovery: restart all running bots after server restart
	fmt.Println("🔧 About to start auto-recovery...")
	if err := recoverRunningBots(tradingBotRepository, startTradingBotUseCase); err != nil {
		fmt.Printf("⚠️ Auto-recovery completed with some errors: %v\n", err)
	} else {
		fmt.Println("🔧 Auto-recovery finished successfully")
	}

	stopTradingBotUseCase := usecase.NewStopTradingBotUseCase(tradingBotRepository)
	stopTradingBotController := api.NewStopTradingBotController(stopTradingBotUseCase)
	http.HandleFunc("/api/v1/trading/stop", authMiddleware.RequireAuth(stopTradingBotController.Handle))

	backtestStrategyUseCase := usecase.NewBacktestStrategyUseCase()
	historicalDataService := external.NewBinanceHistoricalDataService(binanceWrapper)
	backtestStrategyController := api.NewBacktestStrategyController(backtestStrategyUseCase, historicalDataService)
	http.HandleFunc("/api/v1/trading/backtest", authMiddleware.RequireAuth(backtestStrategyController.Handle))

	listTradingLogsUseCase := usecase.NewListTradingLogsUseCase(decisionLogRepository, tradingBotRepository)
	tradingLogsController := api.NewTradingLogsController(listTradingLogsUseCase)
	http.HandleFunc("/api/v1/trading/logs", authMiddleware.RequireAuth(tradingLogsController.ListLogs))

	// Sentiment Analysis System
	sentimentSuggestionRepository := infraRepository.NewSentimentSuggestionRepositoryDatabase(dbConnection.DB)
	generateSentimentUseCase := usecase.NewGenerateSentimentSuggestionUseCase(sentimentSuggestionRepository)
	listSentimentUseCase := usecase.NewListSentimentSuggestionsUseCase(sentimentSuggestionRepository)
	approveSentimentUseCase := usecase.NewApproveSentimentSuggestionUseCase(sentimentSuggestionRepository, tradingBotRepository)
	sentimentController := controller.NewSentimentController(generateSentimentUseCase, listSentimentUseCase, approveSentimentUseCase)
	
	// Sentiment API endpoints
	http.HandleFunc("/api/v1/sentiment/generate", authMiddleware.RequireAuth(sentimentController.GenerateSuggestion))
	http.HandleFunc("/api/v1/sentiment/suggestions", authMiddleware.RequireAuth(sentimentController.ListSuggestions))
	http.HandleFunc("/api/v1/sentiment/approve", authMiddleware.RequireAuth(sentimentController.ApproveSuggestion))
	http.HandleFunc("/api/v1/sentiment/analytics", authMiddleware.RequireAuth(sentimentController.GetAnalytics))
	http.HandleFunc("/api/v1/sentiment/health", sentimentController.HealthCheck) // Public health check

	// Telegram test endpoints
	telegramTestController := api.NewTelegramTestController(telegramService)
	http.HandleFunc("/api/v1/telegram/test", telegramTestController.SendOi) // Temporary public for demo
	http.HandleFunc("/api/v1/telegram/status", telegramTestController.Status) // Public status check

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
