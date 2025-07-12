package main

import (
	"crypgo-machine/src/application/usecase"
	"crypgo-machine/src/infra/api"
	"crypgo-machine/src/infra/database"
	"crypgo-machine/src/infra/external"
	"crypgo-machine/src/infra/notification"
	"crypgo-machine/src/infra/queue"
	"crypgo-machine/src/infra/repository"
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

	tradingBotRepository := repository.NewTradingBotRepositoryDatabase(dbConnection.DB)
	rabbit, err := queue.NewRabbitQMAdapter(os.Getenv("RABBIT_MQ_URL"))
	if err != nil {
		log.Fatal(err)
	}
	decisionLogRepository := repository.NewTradingDecisionLogRepositoryDatabase(dbConnection.DB)

	notificationConsumer := notification.NewEmailNotificationConsumer(rabbit, "trading_bot", "email.notification.queue")
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
	startTradingBotUseCase := usecase.NewStartTradingBotUseCase(tradingBotRepository, decisionLogRepository, binanceWrapper)
	startTradingBotController := api.NewStartTradingBotController(startTradingBotUseCase)
	http.HandleFunc("/api/v1/trading/start", startTradingBotController.Handle)

	stopTradingBotUseCase := usecase.NewStopTradingBotUseCase(tradingBotRepository)
	stopTradingBotController := api.NewStopTradingBotController(stopTradingBotUseCase)
	http.HandleFunc("/api/v1/trading/stop", stopTradingBotController.Handle)

	backtestStrategyUseCase := usecase.NewBacktestStrategyUseCase()
	historicalDataService := external.NewBinanceHistoricalDataService(binanceWrapper)
	backtestStrategyController := api.NewBacktestStrategyController(backtestStrategyUseCase, historicalDataService)
	http.HandleFunc("/api/v1/trading/backtest", backtestStrategyController.Handle)

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
