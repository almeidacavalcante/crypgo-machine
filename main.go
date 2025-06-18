package main

import (
	"crypgo-machine/src/application/usecase"
	"crypgo-machine/src/infra/api"
	"crypgo-machine/src/infra/database"
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

	tradeBotRepository := repository.NewTradeBotRepositoryDatabase(dbConnection.DB)
	createTradingBotUseCase := usecase.NewCreateTradingBotUseCase(tradeBotRepository, *client)

	createTradingBotController := api.NewCreateTradingBotController(createTradingBotUseCase)
	http.HandleFunc("/api/v1/trading/create_trading_bot", createTradingBotController.CreateBot)

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
