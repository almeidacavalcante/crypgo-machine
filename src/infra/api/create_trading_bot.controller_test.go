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

// newInMemoryCreateController builds a controller backed by the in-memory repository.
// It does not need the test database.
func newInMemoryCreateController() (*api.CreateTradingBotController, *repository.TradeBotRepositoryInMemory) {
	tradeBotRepo := repository.NewTradeBotRepositoryInMemory()
	useCase := usecase.NewCreateTradingBotUseCase(tradeBotRepo, binance.Client{}, &MockMessageBroker{}, "test-exchange")
	return api.NewCreateTradingBotController(useCase), tradeBotRepo
}

func postCreateTradingBot(t *testing.T, controller *api.CreateTradingBotController, body interface{}) (int, string) {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}
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
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(respBody)
}

func validRSIBody() map[string]interface{} {
	return map[string]interface{}{
		"symbol":                   "BTCUSDT",
		"quantity":                 0.001,
		"strategy":                 "RSI",
		"interval_seconds":         300,
		"initial_capital":          1000.0,
		"trade_amount":             100.0,
		"currency":                 "USDT",
		"trading_fees":             0.1,
		"minimum_profit_threshold": 1.0,
		"use_fixed_quantity":       true,
		"params": map[string]interface{}{
			"Period":              14,
			"OversoldThreshold":   25.0,
			"OverboughtThreshold": 75.0,
		},
	}
}

func getOnlyBot(t *testing.T, repo *repository.TradeBotRepositoryInMemory) map[string]interface{} {
	t.Helper()
	bots, err := repo.GetAllTradingBots()
	if err != nil {
		t.Fatalf("failed to list bots: %v", err)
	}
	if len(bots) != 1 {
		t.Fatalf("expected 1 saved bot, got %d", len(bots))
	}
	bot := bots[0]
	return map[string]interface{}{
		"strategy":           bot.GetStrategy().GetName(),
		"params":             bot.GetStrategy().GetParams(),
		"use_fixed_quantity": bot.GetUseFixedQuantity(),
	}
}

func TestCreateTradingBotController_RSI_Success(t *testing.T) {
	controller, repo := newInMemoryCreateController()

	status, respBody := postCreateTradingBot(t, controller, validRSIBody())
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Response: %s", status, respBody)
	}

	saved := getOnlyBot(t, repo)
	if saved["strategy"] != "RSI" {
		t.Errorf("expected strategy RSI, got %v", saved["strategy"])
	}
	params := saved["params"].(map[string]interface{})
	if params["Period"] != 14 {
		t.Errorf("expected Period 14, got %v", params["Period"])
	}
	if params["OversoldThreshold"] != 25.0 || params["OverboughtThreshold"] != 75.0 {
		t.Errorf("expected thresholds 25/75, got %v/%v", params["OversoldThreshold"], params["OverboughtThreshold"])
	}
}

func TestCreateTradingBotController_RSI_OmittedThresholdsUseDefaults(t *testing.T) {
	controller, repo := newInMemoryCreateController()

	body := validRSIBody()
	body["params"] = map[string]interface{}{"Period": 14}

	status, respBody := postCreateTradingBot(t, controller, body)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Response: %s", status, respBody)
	}

	params := getOnlyBot(t, repo)["params"].(map[string]interface{})
	if params["OversoldThreshold"] != 30.0 || params["OverboughtThreshold"] != 70.0 {
		t.Errorf("expected default thresholds 30/70, got %v/%v", params["OversoldThreshold"], params["OverboughtThreshold"])
	}
}

func TestCreateTradingBotController_RSI_InvalidParams(t *testing.T) {
	controller, repo := newInMemoryCreateController()

	body := validRSIBody()
	body["params"] = map[string]interface{}{"Period": 0}

	status, _ := postCreateTradingBot(t, controller, body)
	if status != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid RSI period, got %d", status)
	}
	if bots, _ := repo.GetAllTradingBots(); len(bots) != 0 {
		t.Errorf("expected no bot to be saved, got %d", len(bots))
	}
}

func TestCreateTradingBotController_RSI_MalformedParams(t *testing.T) {
	controller, _ := newInMemoryCreateController()

	body := validRSIBody()
	body["params"] = map[string]interface{}{"Period": "fourteen"}

	status, _ := postCreateTradingBot(t, controller, body)
	if status != http.StatusBadRequest {
		t.Errorf("expected status 400 for malformed RSI params, got %d", status)
	}
}

func TestCreateTradingBotController_UseFixedQuantityIsPersisted(t *testing.T) {
	for _, useFixed := range []bool{true, false} {
		controller, repo := newInMemoryCreateController()

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
			"use_fixed_quantity":       useFixed,
			"params": map[string]interface{}{
				"FastWindow": 7,
				"SlowWindow": 21,
			},
		}

		status, respBody := postCreateTradingBot(t, controller, body)
		if status != http.StatusCreated {
			t.Fatalf("use_fixed_quantity=%v: expected status 201, got %d. Response: %s", useFixed, status, respBody)
		}
		if got := getOnlyBot(t, repo)["use_fixed_quantity"]; got != useFixed {
			t.Errorf("expected use_fixed_quantity %v to be persisted, got %v", useFixed, got)
		}
	}
}

func TestCreateTradingBotController_ExampleRSIRequestFile(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "example_rsi_bot_request.json"))
	if err != nil {
		t.Fatalf("failed to read example_rsi_bot_request.json: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("example_rsi_bot_request.json is not valid JSON: %v", err)
	}
	if _, ok := body["params"]; !ok {
		t.Fatalf("example_rsi_bot_request.json must use the \"params\" key read by the controller")
	}

	controller, repo := newInMemoryCreateController()
	status, respBody := postCreateTradingBot(t, controller, body)
	if status != http.StatusCreated {
		t.Fatalf("expected status 201 for example RSI request, got %d. Response: %s", status, respBody)
	}
	saved := getOnlyBot(t, repo)
	if saved["strategy"] != "RSI" {
		t.Errorf("expected strategy RSI, got %v", saved["strategy"])
	}
	if saved["use_fixed_quantity"] != true {
		t.Errorf("expected use_fixed_quantity true from example file, got %v", saved["use_fixed_quantity"])
	}
}
