package api_test

import (
	"bytes"
	"crypgo-machine/src/application/usecase"
	"crypgo-machine/src/infra/api"
	"crypgo-machine/src/infra/database"
	"crypgo-machine/src/infra/repository"
	"encoding/json"
	"github.com/adshao/go-binance/v2"
	"github.com/joho/godotenv"
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
	db, err := database.NewDatabaseConnection(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}
	return db
}

func cleanupTradeBot(t *testing.T, db *database.Connection, symbol string) {
	t.Helper()
	_, err := db.DB.Exec("DELETE FROM trade_bots WHERE symbol = $1", symbol)
	if err != nil {
		t.Fatalf("failed to cleanup: %v", err)
	}
}

func TestCreateTradingBotController_MovingAverage_Success(t *testing.T) {
	// Setup DB and dependencies

	dbConn := setupTestDB(t)
	defer dbConn.DB.Close()
	tradeBotRepo := repository.NewTradeBotRepositoryInMemory()
	//tradeBotRepo := repository.NewTradeBotRepositoryDatabase(dbConn.DB)
	useCase := usecase.NewCreateTradingBotUseCase(tradeBotRepo, binance.Client{}) // Client não é usado
	controller := api.NewCreateTradingBotController(useCase)

	// Prepare request body
	body := map[string]interface{}{
		"symbol":   "SOLBRL",
		"quantity": 10,
		"strategy": "MovingAverage",
		"params": map[string]interface{}{
			"FastWindow": 7,
			"SlowWindow": 21,
		},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trading/create_trading_bot", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	controller.CreateBot(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	// Limpa após o teste
	cleanupTradeBot(t, dbConn, "SOLBRL")
}

func TestCreateTradingBotController_InvalidParams(t *testing.T) {
	dbConn := setupTestDB(t)
	defer dbConn.DB.Close()
	tradeBotRepo := repository.NewTradeBotRepositoryDatabase(dbConn.DB)
	useCase := usecase.NewCreateTradingBotUseCase(tradeBotRepo, binance.Client{})
	controller := api.NewCreateTradingBotController(useCase)

	body := map[string]interface{}{
		"symbol":   "SOLBRL",
		"quantity": 1,
		"strategy": "MovingAverage",
		"params": map[string]interface{}{
			"FastWindow": 7,
			// Falta SlowWindow (inválido)
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/bots", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	controller.CreateBot(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
	cleanupTradeBot(t, dbConn, "SOLBRL")
}
