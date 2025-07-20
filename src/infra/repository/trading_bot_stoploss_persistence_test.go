package repository

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/service"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/database"
	"database/sql"
	"os"
	"testing"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *sql.DB {
	err := godotenv.Load("../../../.env")
	if err != nil {
		t.Skip("Skipping database test: .env file not found")
	}

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	connection, err := database.NewDatabaseConnection(host, port, user, password, dbname)
	if err != nil {
		t.Skipf("Skipping database test: failed to connect to database: %v", err)
	}

	return connection.DB
}

func cleanupTestBot(t *testing.T, db *sql.DB, botID string) {
	// Delete trading decision logs first (foreign key constraint)
	_, err := db.Exec("DELETE FROM trading_decision_logs WHERE trading_bot_id = $1", botID)
	if err != nil {
		t.Logf("Warning: failed to cleanup trading decision logs: %v", err)
	}

	// Then delete the bot
	_, err = db.Exec("DELETE FROM trade_bots WHERE id = $1", botID)
	if err != nil {
		t.Logf("Warning: failed to cleanup test bot: %v", err)
	}
}

func TestTradingBotStoplossPersistence_MovingAverage(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTradingBotRepositoryDatabase(db)

	// Create Moving Average strategy with stoploss
	symbol, _ := vo.NewSymbol("BTCUSDT")
	
	// Use the factory to create the strategy with stoploss
	strategyParams := service.MovingAverageParams{
		FastWindow:        5,
		SlowWindow:        20,
		StoplossThreshold: 7.5,
	}
	
	strategy, err := service.NewTradeStrategyFactory("MovingAverage", strategyParams)
	if err != nil {
		t.Fatalf("Failed to create strategy: %v", err)
	}

	// Create trading bot
	bot := entity.NewTradingBot(
		symbol,
		0.001,
		strategy,
		300,
		10000.0,
		1000.0,
		"USDT",
		0.1,
		2.0,
		true,
	)

	botID := string(bot.Id.GetValue())
	defer cleanupTestBot(t, db, botID)

	// Save the bot
	err = repo.Save(bot)
	if err != nil {
		t.Fatalf("Failed to save bot: %v", err)
	}

	// Retrieve the bot
	retrievedBot, err := repo.GetTradeByID(botID)
	if err != nil {
		t.Fatalf("Failed to retrieve bot: %v", err)
	}

	if retrievedBot == nil {
		t.Fatal("Retrieved bot is nil")
	}

	// Verify strategy is MovingAverage with stoploss
	if retrievedBot.GetStrategy().GetName() != "MovingAverage" {
		t.Errorf("Expected strategy name 'MovingAverage', got '%s'", retrievedBot.GetStrategy().GetName())
	}

	params := retrievedBot.GetStrategy().GetParams()
	
	// Check stoploss threshold
	stoplossThreshold, exists := params["StoplossThreshold"]
	if !exists {
		t.Error("StoplossThreshold parameter not found")
	}

	if stoplossThreshold != 7.5 {
		t.Errorf("Expected StoplossThreshold 7.5, got %v", stoplossThreshold)
	}

	// Check other parameters
	if params["FastWindow"] != 5 {
		t.Errorf("Expected FastWindow 5, got %v", params["FastWindow"])
	}

	if params["SlowWindow"] != 20 {
		t.Errorf("Expected SlowWindow 20, got %v", params["SlowWindow"])
	}

	t.Logf("✅ MovingAverage with stoploss %.1f%% persisted and retrieved successfully", stoplossThreshold)
}

func TestTradingBotStoplossPersistence_RSI(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTradingBotRepositoryDatabase(db)

	// Create RSI strategy with stoploss
	symbol, _ := vo.NewSymbol("ETHUSDT")
	
	// Use the factory to create the strategy with stoploss
	strategyParams := service.RSIParams{
		Period:              14,
		OversoldThreshold:   25.0,
		OverboughtThreshold: 75.0,
		StoplossThreshold:   5.0,
	}
	
	strategy, err := service.NewTradeStrategyFactory("RSI", strategyParams)
	if err != nil {
		t.Fatalf("Failed to create RSI strategy: %v", err)
	}

	// Create trading bot
	bot := entity.NewTradingBot(
		symbol,
		0.01,
		strategy,
		300,
		5000.0,
		500.0,
		"USDT",
		0.1,
		1.5,
		true,
	)

	botID := string(bot.Id.GetValue())
	defer cleanupTestBot(t, db, botID)

	// Save the bot
	err = repo.Save(bot)
	if err != nil {
		t.Fatalf("Failed to save RSI bot: %v", err)
	}

	// Retrieve the bot
	retrievedBot, err := repo.GetTradeByID(botID)
	if err != nil {
		t.Fatalf("Failed to retrieve RSI bot: %v", err)
	}

	if retrievedBot == nil {
		t.Fatal("Retrieved RSI bot is nil")
	}

	// Verify strategy is RSI with stoploss
	if retrievedBot.GetStrategy().GetName() != "RSI" {
		t.Errorf("Expected strategy name 'RSI', got '%s'", retrievedBot.GetStrategy().GetName())
	}

	params := retrievedBot.GetStrategy().GetParams()
	
	// Check stoploss threshold
	stoplossThreshold, exists := params["StoplossThreshold"]
	if !exists {
		t.Error("StoplossThreshold parameter not found in RSI strategy")
	}

	if stoplossThreshold != 5.0 {
		t.Errorf("Expected RSI StoplossThreshold 5.0, got %v", stoplossThreshold)
	}

	// Check other RSI parameters
	if params["Period"] != 14 {
		t.Errorf("Expected Period 14, got %v", params["Period"])
	}

	if params["OversoldThreshold"] != 25.0 {
		t.Errorf("Expected OversoldThreshold 25.0, got %v", params["OversoldThreshold"])
	}

	if params["OverboughtThreshold"] != 75.0 {
		t.Errorf("Expected OverboughtThreshold 75.0, got %v", params["OverboughtThreshold"])
	}

	t.Logf("✅ RSI with stoploss %.1f%% persisted and retrieved successfully", stoplossThreshold)
}

func TestTradingBotPersistence_WithoutStoploss(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTradingBotRepositoryDatabase(db)

	// Create Moving Average strategy WITHOUT stoploss
	symbol, _ := vo.NewSymbol("ADAUSDT")
	
	strategyParams := service.MovingAverageParams{
		FastWindow:        3,
		SlowWindow:        40,
		StoplossThreshold: 0.0, // No stoploss
	}
	
	strategy, err := service.NewTradeStrategyFactory("MovingAverage", strategyParams)
	if err != nil {
		t.Fatalf("Failed to create strategy without stoploss: %v", err)
	}

	// Create trading bot
	bot := entity.NewTradingBot(
		symbol,
		0.5,
		strategy,
		300,
		1000.0,
		100.0,
		"USDT",
		0.1,
		3.0,
		false,
	)

	botID := string(bot.Id.GetValue())
	defer cleanupTestBot(t, db, botID)

	// Save the bot
	err = repo.Save(bot)
	if err != nil {
		t.Fatalf("Failed to save bot without stoploss: %v", err)
	}

	// Retrieve the bot
	retrievedBot, err := repo.GetTradeByID(botID)
	if err != nil {
		t.Fatalf("Failed to retrieve bot without stoploss: %v", err)
	}

	if retrievedBot == nil {
		t.Fatal("Retrieved bot without stoploss is nil")
	}

	// Verify strategy parameters
	params := retrievedBot.GetStrategy().GetParams()
	
	// Check stoploss threshold is 0 or not present
	stoplossThreshold, _ := params["StoplossThreshold"]
	if stoplossThreshold != 0.0 {
		t.Errorf("Expected no stoploss (0.0), got %v", stoplossThreshold)
	}

	t.Log("✅ Strategy without stoploss persisted correctly")
}

func TestMultipleStrategiesStoplossPersistence(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewTradingBotRepositoryDatabase(db)

	// Test data: different strategies with different stoploss values
	testCases := []struct {
		name              string
		symbol            string
		strategyType      string
		params            interface{}
		expectedStoploss  float64
	}{
		{
			name:         "MA_Stoploss_3%",
			symbol:       "SOLUSDT",
			strategyType: "MovingAverage", 
			params: service.MovingAverageParams{
				FastWindow:        7,
				SlowWindow:        25,
				StoplossThreshold: 3.0,
			},
			expectedStoploss: 3.0,
		},
		{
			name:         "RSI_Stoploss_10%",
			symbol:       "DOTUSDT",
			strategyType: "RSI",
			params: service.RSIParams{
				Period:              21,
				OversoldThreshold:   20.0,
				OverboughtThreshold: 80.0,
				StoplossThreshold:   10.0,
			},
			expectedStoploss: 10.0,
		},
		{
			name:         "MA_No_Stoploss",
			symbol:       "LINKUSDT",
			strategyType: "MovingAverage",
			params: service.MovingAverageParams{
				FastWindow:        5,
				SlowWindow:        15,
				StoplossThreshold: 0.0,
			},
			expectedStoploss: 0.0,
		},
	}

	var createdBotIDs []string

	// Create and save all bots
	for _, tc := range testCases {
		symbol, _ := vo.NewSymbol(tc.symbol)
		
		strategy, err := service.NewTradeStrategyFactory(tc.strategyType, tc.params)
		if err != nil {
			t.Fatalf("Failed to create strategy %s: %v", tc.name, err)
		}

		bot := entity.NewTradingBot(
			symbol,
			0.1,
			strategy,
			300,
			2000.0,
			200.0,
			"USDT",
			0.1,
			2.0,
			true,
		)

		botID := string(bot.Id.GetValue())
		createdBotIDs = append(createdBotIDs, botID)

		err = repo.Save(bot)
		if err != nil {
			t.Fatalf("Failed to save bot %s: %v", tc.name, err)
		}
	}

	// Cleanup at the end
	defer func() {
		for _, botID := range createdBotIDs {
			cleanupTestBot(t, db, botID)
		}
	}()

	// Retrieve and verify all bots
	for i, tc := range testCases {
		botID := createdBotIDs[i]
		
		retrievedBot, err := repo.GetTradeByID(botID)
		if err != nil {
			t.Fatalf("Failed to retrieve bot %s: %v", tc.name, err)
		}

		if retrievedBot == nil {
			t.Fatalf("Retrieved bot %s is nil", tc.name)
		}

		params := retrievedBot.GetStrategy().GetParams()
		stoplossThreshold, _ := params["StoplossThreshold"]

		if stoplossThreshold != tc.expectedStoploss {
			t.Errorf("Bot %s: Expected stoploss %.1f, got %v", tc.name, tc.expectedStoploss, stoplossThreshold)
		} else {
			t.Logf("✅ Bot %s: Stoploss %.1f%% persisted correctly", tc.name, tc.expectedStoploss)
		}
	}
}