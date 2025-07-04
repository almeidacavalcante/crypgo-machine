package api_test

import (
	"bytes"
	"crypgo-machine/src/application/usecase"
	"crypgo-machine/src/infra/api"
	"crypgo-machine/src/infra/database"
	"crypgo-machine/src/infra/queue"
	"crypgo-machine/src/infra/repository"
	"encoding/json"
	"github.com/adshao/go-binance/v2"
	"github.com/joho/godotenv"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func mustLoadDotenv(t *testing.T) {
	t.Helper()
	cur, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}

	// Sobe até 8 níveis procurando o .env
	for i := 0; i < 8; i++ {
		envPath := filepath.Join(cur, ".env")
		if _, err := os.Stat(envPath); err == nil {
			if err := godotenv.Load(envPath); err != nil {
				t.Fatalf("failed to load .env: %v", err)
			}
			return
		}
		cur = filepath.Dir(cur)
	}
	t.Fatalf(".env file not found in parent directories")
}

func setupTestDB(t *testing.T) *database.Connection {
	t.Helper()
	mustLoadDotenv(t) // Isso garante que SEMPRE carrega o .env certo
	// Use test database configuration
	db, err := database.NewDatabaseConnection(
		"localhost",
		"5433",
		"crypgo_test",
		"crypgo_test",
		"crypgo_machine_test",
	)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}
	return db
}

func cleanupTradeBot(t *testing.T, db *database.Connection, symbol string) {
	t.Helper()
	// First delete decision logs, then trade bots due to foreign key constraint
	_, err := db.DB.Exec(`
		DELETE FROM trading_decision_logs 
		WHERE trading_bot_id IN (
			SELECT id FROM trade_bots WHERE symbol = $1
		)
	`, symbol)
	if err != nil {
		t.Logf("Warning: failed to cleanup decision logs: %v", err)
	}

	_, err = db.DB.Exec("DELETE FROM trade_bots WHERE symbol = $1", symbol)
	if err != nil {
		t.Logf("Warning: failed to cleanup trade bots: %v", err)
	}
}

// MockMessageBroker implements queue.MessageBroker for testing
type MockMessageBroker struct{}

func (m *MockMessageBroker) Publish(exchangeName string, message queue.Message) error {
	// Mock implementation - does nothing
	return nil
}

func (m *MockMessageBroker) Subscribe(exchangeName string, queueName string, routingKeys []string, handler func(msg queue.Message) error) error {
	// Mock implementation - does nothing
	return nil
}

func (m *MockMessageBroker) Close() error {
	// Mock implementation - does nothing
	return nil
}

func TestCreateTradingBotController_MovingAverage_Success(t *testing.T) {
	// Setup DB and dependencies

	dbConn := setupTestDB(t)
	defer func() {
		if err := dbConn.DB.Close(); err != nil {
			t.Logf("Error closing DB: %v", err)
		}
	}()
	tradeBotRepo := repository.NewTradeBotRepositoryInMemory()
	//tradeBotRepo := repository.NewTradingBotRepositoryDatabase(dbConn.DB)
	mockMessageBroker := &MockMessageBroker{}
	useCase := usecase.NewCreateTradingBotUseCase(tradeBotRepo, binance.Client{}, mockMessageBroker, "test-exchange")
	controller := api.NewCreateTradingBotController(useCase)

	// Prepare request body
	body := map[string]interface{}{
		"symbol":                   "SOLBRL",
		"quantity":                 10,
		"strategy":                 "MovingAverage",
		"interval_seconds":         3600,
		"initial_capital":          10000.0,
		"trade_amount":             4000.0,
		"currency":                 "BRL",
		"trading_fees":             0.001,
		"minimum_profit_threshold": 5.0,
		"params": map[string]interface{}{
			"FastWindow": 7,
			"SlowWindow": 21,
		},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trading/create_trading_bot", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	controller.Handle(rec, req)
	resp := rec.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusCreated {
		// Read response body to see the error
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 201, got %d. Response: %s", resp.StatusCode, string(body))
	}

	// Limpa após o teste
	cleanupTradeBot(t, dbConn, "SOLBRL")
}

func TestCreateTradingBotController_InvalidParams(t *testing.T) {
	dbConn := setupTestDB(t)
	defer func() {
		if err := dbConn.DB.Close(); err != nil {
			t.Logf("Error closing DB: %v", err)
		}
	}()
	tradeBotRepo := repository.NewTradeBotRepositoryInMemory()
	mockMessageBroker := &MockMessageBroker{}
	useCase := usecase.NewCreateTradingBotUseCase(tradeBotRepo, binance.Client{}, mockMessageBroker, "test-exchange")
	controller := api.NewCreateTradingBotController(useCase)

	body := map[string]interface{}{
		"symbol":                   "SOLBRL",
		"quantity":                 1,
		"strategy":                 "MovingAverage",
		"interval_seconds":         3600,
		"initial_capital":          10000.0,
		"trade_amount":             4000.0,
		"currency":                 "BRL",
		"trading_fees":             0.001,
		"minimum_profit_threshold": 5.0,
		"params": map[string]interface{}{
			"FastWindow": 7,
			// Falta SlowWindow (inválido)
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/bots", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	controller.Handle(rec, req)
	resp := rec.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
	cleanupTradeBot(t, dbConn, "SOLBRL")
}

func TestCreateTradingBotController_SmallWindowDifference(t *testing.T) {
	dbConn := setupTestDB(t)
	defer func() {
		if err := dbConn.DB.Close(); err != nil {
			t.Logf("Error closing DB: %v", err)
		}
	}()
	tradeBotRepo := repository.NewTradeBotRepositoryInMemory()
	mockMessageBroker := &MockMessageBroker{}
	useCase := usecase.NewCreateTradingBotUseCase(tradeBotRepo, binance.Client{}, mockMessageBroker, "test-exchange")
	controller := api.NewCreateTradingBotController(useCase)

	body := map[string]interface{}{
		"symbol":                   "ETHBRL",
		"quantity":                 5,
		"strategy":                 "MovingAverage",
		"interval_seconds":         3600,
		"initial_capital":          10000.0,
		"trade_amount":             4000.0,
		"currency":                 "BRL",
		"trading_fees":             0.001,
		"minimum_profit_threshold": 5.0,
		"params": map[string]interface{}{
			"FastWindow": 10,
			"SlowWindow": 11,
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trading/create_trading_bot", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	controller.Handle(rec, req)
	resp := rec.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for small window difference, got %d", resp.StatusCode)
	}

	cleanupTradeBot(t, dbConn, "ETHBRL")
}

func TestCreateTradingBotController_MissingFinancialFields(t *testing.T) {
	dbConn := setupTestDB(t)
	defer func() {
		if err := dbConn.DB.Close(); err != nil {
			t.Logf("Error closing DB: %v", err)
		}
	}()
	tradeBotRepo := repository.NewTradeBotRepositoryInMemory()
	mockMessageBroker := &MockMessageBroker{}
	useCase := usecase.NewCreateTradingBotUseCase(tradeBotRepo, binance.Client{}, mockMessageBroker, "test-exchange")
	controller := api.NewCreateTradingBotController(useCase)

	body := map[string]interface{}{
		"symbol":   "SOLBRL",
		"quantity": 10,
		"strategy": "MovingAverage",
		"params": map[string]interface{}{
			"FastWindow": 7,
			"SlowWindow": 21,
		},
		// Missing financial fields
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trading/create_trading_bot", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	controller.Handle(rec, req)
	resp := rec.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing financial fields, got %d", resp.StatusCode)
	}

	cleanupTradeBot(t, dbConn, "SOLBRL")
}

func TestCreateTradingBotController_InvalidInitialCapital(t *testing.T) {
	dbConn := setupTestDB(t)
	defer func() {
		if err := dbConn.DB.Close(); err != nil {
			t.Logf("Error closing DB: %v", err)
		}
	}()
	tradeBotRepo := repository.NewTradeBotRepositoryInMemory()
	mockMessageBroker := &MockMessageBroker{}
	useCase := usecase.NewCreateTradingBotUseCase(tradeBotRepo, binance.Client{}, mockMessageBroker, "test-exchange")
	controller := api.NewCreateTradingBotController(useCase)

	body := map[string]interface{}{
		"symbol":                   "SOLBRL",
		"quantity":                 10,
		"strategy":                 "MovingAverage",
		"interval_seconds":         3600,
		"initial_capital":          0, // Invalid - should be greater than zero
		"trade_amount":             4000.0,
		"currency":                 "BRL",
		"trading_fees":             0.001,
		"minimum_profit_threshold": 5.0,
		"params": map[string]interface{}{
			"FastWindow": 7,
			"SlowWindow": 21,
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trading/create_trading_bot", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	controller.Handle(rec, req)
	resp := rec.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid initial capital, got %d", resp.StatusCode)
	}

	cleanupTradeBot(t, dbConn, "SOLBRL")
}
