package usecase

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"crypgo-machine/src/infra/repository"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func setupStartTradingBotUseCase() (*StartTradingBotUseCase, *repository.TradeBotRepositoryInMemory, *repository.TradingDecisionLogRepositoryInMemory, *external.BinanceClientFake) {
	tradingBotRepo := repository.NewTradeBotRepositoryInMemory()
	decisionLogRepo := repository.NewTradingDecisionLogRepositoryInMemory()
	binanceClient := external.NewBinanceClientFake()

	useCase := NewStartTradingBotUseCase(tradingBotRepo, decisionLogRepo, binanceClient)

	return useCase, tradingBotRepo, decisionLogRepo, binanceClient
}

func createTestTradingBot() *entity.TradingBot {
	symbol, _ := vo.NewSymbol("BTCUSDT")
	strategy := entity.NewMovingAverageStrategy(7, 40)
	bot := entity.NewTradingBot(symbol, 0.001, strategy, 60, 10000.0, 1000.0, "USDT", 0.001, 0.0, true)
	return bot
}

func TestStartTradingBotUseCase_Execute_Success(t *testing.T) {
	useCase, tradingBotRepo, _, _ := setupStartTradingBotUseCase()

	// Create and save a test trading bot
	bot := createTestTradingBot()
	err := tradingBotRepo.Save(bot)
	if err != nil {
		t.Fatalf("Failed to save bot: %v", err)
	}

	// Execute the use case
	input := InputStartTradingBot{
		TradingBotId: bot.Id.GetValue(),
	}

	err = useCase.Execute(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify bot status changed to running
	updatedBot, err := tradingBotRepo.GetTradeByID(bot.Id.GetValue())
	if err != nil {
		t.Fatalf("Failed to get updated bot: %v", err)
	}

	if updatedBot.GetStatus() != entity.StatusRunning {
		t.Errorf("Expected bot status to be %v, got %v", entity.StatusRunning, updatedBot.GetStatus())
	}
}

func TestStartTradingBotUseCase_Execute_BotNotFound(t *testing.T) {
	useCase, _, _, _ := setupStartTradingBotUseCase()

	input := InputStartTradingBot{
		TradingBotId: "non-existent-id",
	}

	err := useCase.Execute(input)
	if err == nil {
		t.Fatal("Expected error for non-existent bot, got none")
	}

	expectedError := "trading bot not found"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestStartTradingBotUseCase_Execute_BotAlreadyRunning(t *testing.T) {
	useCase, tradingBotRepo, _, _ := setupStartTradingBotUseCase()

	// Create a bot that's already running
	bot := createTestTradingBot()
	_ = bot.Start() // Set status to running
	err := tradingBotRepo.Save(bot)
	if err != nil {
		t.Fatalf("Failed to save bot: %v", err)
	}

	input := InputStartTradingBot{
		TradingBotId: bot.Id.GetValue(),
	}

	err = useCase.Execute(input)
	if err == nil {
		t.Fatal("Expected error for already running bot, got none")
	}
}

func TestStartTradingBotUseCase_WhipsawScenario(t *testing.T) {
	useCase, tradingBotRepo, _, binanceClient := setupStartTradingBotUseCase()

	// Setup whipsaw klines that should be filtered out
	whipsawKlines := external.CreateWhipsawKlines()
	binanceClient.SetPredefinedKlines(whipsawKlines)

	// Create and save a test trading bot
	bot := createTestTradingBot()
	err := tradingBotRepo.Save(bot)
	if err != nil {
		t.Fatalf("Failed to save bot: %v", err)
	}

	// Start the bot
	input := InputStartTradingBot{
		TradingBotId: bot.Id.GetValue(),
	}

	err = useCase.Execute(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Simulate the strategy loop execution manually (since we can't wait for goroutine)
	klines, err := useCase.getMarketData(bot.GetSymbol().GetValue(), 300)
	if err != nil {
		t.Fatalf("Failed to get market data: %v", err)
	}

	strategy := bot.GetStrategy()
	analysisResult := strategy.Decide(klines, bot)

	// The whipsaw test data with strong uptrend should generate HOLD (waiting for dip)
	// This is expected behavior - the strategy waits for fast MA to go below slow MA
	if analysisResult.Decision != entity.Hold {
		t.Errorf("Expected HOLD decision with uptrend (waiting for dip), got %v", analysisResult.Decision)
	}

	// Verify the reason indicates waiting for dip
	reason, ok := analysisResult.AnalysisData["reason"].(string)
	if !ok {
		t.Fatal("Expected reason in analysis data")
	}

	if reason != "fast_above_slow_wait_for_dip" {
		t.Errorf("Expected wait for dip reason, got: %s", reason)
	}
}

func TestStartTradingBotUseCase_StrongTrendScenario(t *testing.T) {
	useCase, tradingBotRepo, _, binanceClient := setupStartTradingBotUseCase()

	// Setup strong trend klines that should generate signals
	strongTrendKlines := external.CreateStrongTrendKlines()
	binanceClient.SetPredefinedKlines(strongTrendKlines)

	// Create and save a test trading bot
	bot := createTestTradingBot()
	err := tradingBotRepo.Save(bot)
	if err != nil {
		t.Fatalf("Failed to save bot: %v", err)
	}

	// Start the bot
	input := InputStartTradingBot{
		TradingBotId: bot.Id.GetValue(),
	}

	err = useCase.Execute(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Simulate the strategy loop execution manually
	klines, err := useCase.getMarketData(bot.GetSymbol().GetValue(), 300)
	if err != nil {
		t.Fatalf("Failed to get market data: %v", err)
	}

	strategy := bot.GetStrategy()
	analysisResult := strategy.Decide(klines, bot)

	// With strong trend data, should generate a HOLD signal (waiting for dip)
	if analysisResult.Decision != entity.Hold {
		t.Errorf("Expected HOLD decision with strong trend (waiting for dip), got %v", analysisResult.Decision)
	}

	// Verify the reason indicates waiting for dip
	reason, ok := analysisResult.AnalysisData["reason"].(string)
	if !ok {
		t.Fatal("Expected reason in analysis data")
	}

	if reason != "fast_above_slow_wait_for_dip" {
		t.Errorf("Expected wait for dip reason, got: %s", reason)
	}
}

func TestStartTradingBotUseCase_DecisionLogCreation(t *testing.T) {
	useCase, tradingBotRepo, decisionLogRepo, _ := setupStartTradingBotUseCase()

	// Create and save a test trading bot
	bot := createTestTradingBot()
	err := tradingBotRepo.Save(bot)
	if err != nil {
		t.Fatalf("Failed to save bot: %v", err)
	}

	// Simulate strategy execution (what happens in the goroutine)
	klines, err := useCase.getMarketData(bot.GetSymbol().GetValue(), 300)
	if err != nil {
		t.Fatalf("Failed to get market data: %v", err)
	}

	strategy := bot.GetStrategy()
	analysisResult := strategy.Decide(klines, bot)

	// Create and save decision log manually (simulating the goroutine)
	currentPrice := klines[len(klines)-1].Close()
	decisionLog := entity.NewTradingDecisionLog(
		bot.Id,
		analysisResult.Decision,
		strategy.GetName(),
		analysisResult.AnalysisData,
		klines,
		currentPrice,
		0.0, // currentPossibleProfit for test
	)

	err = decisionLogRepo.Save(decisionLog)
	if err != nil {
		t.Fatalf("Failed to save decision log: %v", err)
	}

	// Verify decision log was saved
	logs, err := decisionLogRepo.GetByTradingBotId(bot.Id.GetValue())
	if err != nil {
		t.Fatalf("Failed to get decision logs: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 decision log, got %d", len(logs))
	}

	log := logs[0]
	if log.GetStrategyName() != "MovingAverage" {
		t.Errorf("Expected strategy name 'MovingAverage', got '%s'", log.GetStrategyName())
	}

	if len(log.GetMarketData()) == 0 {
		t.Error("Expected market data in decision log")
	}

	if log.GetAnalysisData() == nil {
		t.Error("Expected analysis data in decision log")
	}
}

func TestStartTradingBotUseCase_BinanceErrorHandling(t *testing.T) {
	useCase, tradingBotRepo, _, binanceClient := setupStartTradingBotUseCase()

	// Make Binance client fail
	binanceClient.SetShouldFailKlines(true)

	// Create and save a test trading bot
	bot := createTestTradingBot()
	err := tradingBotRepo.Save(bot)
	if err != nil {
		t.Fatalf("Failed to save bot: %v", err)
	}

	// Try to get market data (simulating what happens in the goroutine)
	_, err = useCase.getMarketData(bot.GetSymbol().GetValue(), 300)
	if err == nil {
		t.Fatal("Expected error from Binance client, got none")
	}

	expectedError := "simulated klines error"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestStartTradingBotUseCase_Execute_InvalidIntervalIsRejectedBeforeRunning(t *testing.T) {
	for _, interval := range []int{0, -60, 120} {
		useCase, tradingBotRepo, _, _ := setupStartTradingBotUseCase()
		broker := &recordingLifecycleBroker{}
		useCase.messageBroker = broker

		symbol, _ := vo.NewSymbol("BTCUSDT")
		strategy := entity.NewMovingAverageStrategy(7, 40)
		bot := entity.NewTradingBot(symbol, 0.001, strategy, interval, 10000.0, 1000.0, "USDT", 0.001, 0.0, true)
		if err := tradingBotRepo.Save(bot); err != nil {
			t.Fatalf("failed to save bot: %v", err)
		}

		err := useCase.Execute(InputStartTradingBot{TradingBotId: bot.Id.GetValue()})
		if err == nil {
			t.Fatalf("interval %d: expected error, got nil", interval)
		}
		if !strings.Contains(err.Error(), "invalid interval_seconds") {
			t.Errorf("interval %d: expected invalid interval_seconds error, got %v", interval, err)
		}

		stored, _ := tradingBotRepo.GetTradeByID(bot.Id.GetValue())
		if stored.GetStatus() != entity.StatusStopped {
			t.Errorf("interval %d: expected bot to stay %s, got %s", interval, entity.StatusStopped, stored.GetStatus())
		}
		if msgs, _ := broker.messagesWithKey("trading_bot.started"); len(msgs) != 0 {
			t.Errorf("interval %d: expected no trading_bot.started event, got %d", interval, len(msgs))
		}
	}
}

func TestStartTradingBotUseCase_Execute_PublishesStartedEvent(t *testing.T) {
	tradingBotRepo := repository.NewTradeBotRepositoryInMemory()
	decisionLogRepo := repository.NewTradingDecisionLogRepositoryInMemory()
	broker := &recordingLifecycleBroker{}
	useCase := NewStartTradingBotUseCaseWithMessaging(tradingBotRepo, decisionLogRepo, external.NewBinanceClientFake(), broker, "test-exchange")

	bot := createTestTradingBot()
	if err := tradingBotRepo.Save(bot); err != nil {
		t.Fatalf("failed to save bot: %v", err)
	}

	if err := useCase.Execute(InputStartTradingBot{TradingBotId: bot.Id.GetValue()}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	msgs, exchanges := broker.messagesWithKey("trading_bot.started")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 trading_bot.started event, got %d", len(msgs))
	}
	if exchanges[0] != "test-exchange" {
		t.Errorf("expected exchange test-exchange, got %s", exchanges[0])
	}
	assertLifecyclePayload(t, msgs[0], bot)
}

func TestStartTradingBotUseCase_Execute_SkipNotification(t *testing.T) {
	tradingBotRepo := repository.NewTradeBotRepositoryInMemory()
	decisionLogRepo := repository.NewTradingDecisionLogRepositoryInMemory()
	broker := &recordingLifecycleBroker{}
	useCase := NewStartTradingBotUseCaseWithMessaging(tradingBotRepo, decisionLogRepo, external.NewBinanceClientFake(), broker, "test-exchange")

	bot := createTestTradingBot()
	if err := tradingBotRepo.Save(bot); err != nil {
		t.Fatalf("failed to save bot: %v", err)
	}

	if err := useCase.Execute(InputStartTradingBot{TradingBotId: bot.Id.GetValue(), SkipNotification: true}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if msgs, _ := broker.messagesWithKey("trading_bot.started"); len(msgs) != 0 {
		t.Errorf("expected no trading_bot.started event, got %d", len(msgs))
	}
}

func TestStartTradingBotUseCase_Execute_PublishFailureDoesNotFailStart(t *testing.T) {
	tradingBotRepo := repository.NewTradeBotRepositoryInMemory()
	decisionLogRepo := repository.NewTradingDecisionLogRepositoryInMemory()
	broker := &recordingLifecycleBroker{publishErr: errors.New("broker down")}
	useCase := NewStartTradingBotUseCaseWithMessaging(tradingBotRepo, decisionLogRepo, external.NewBinanceClientFake(), broker, "test-exchange")

	bot := createTestTradingBot()
	if err := tradingBotRepo.Save(bot); err != nil {
		t.Fatalf("failed to save bot: %v", err)
	}

	if err := useCase.Execute(InputStartTradingBot{TradingBotId: bot.Id.GetValue()}); err != nil {
		t.Fatalf("expected start to succeed despite publish failure, got %v", err)
	}

	stored, _ := tradingBotRepo.GetTradeByID(bot.Id.GetValue())
	if stored.GetStatus() != entity.StatusRunning {
		t.Errorf("expected status %s, got %s", entity.StatusRunning, stored.GetStatus())
	}
}

func TestInputStartTradingBot_SkipNotificationNotDecodedFromJSON(t *testing.T) {
	var input InputStartTradingBot
	if err := json.Unmarshal([]byte(`{"bot_id":"abc","SkipNotification":true,"skip_notification":true}`), &input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.SkipNotification {
		t.Error("expected SkipNotification to be ignored when decoding JSON")
	}
}

// panickingMarketDataSource simulates a bug that panics inside the strategy loop.
type panickingMarketDataSource struct{}

func (panickingMarketDataSource) GetMarketData(symbol string, intervalSeconds int) ([]vo.Kline, error) {
	panic("simulated strategy loop panic")
}

func (panickingMarketDataSource) GetCurrentTime() time.Time {
	return time.Now()
}

// alwaysContinueExecutionContext is a no-op execution context that never asks the loop to stop.
type alwaysContinueExecutionContext struct{}

func (alwaysContinueExecutionContext) ExecuteTrade(decision entity.TradingDecision, bot *entity.TradingBot, currentPrice float64, timestamp time.Time) error {
	return nil
}

func (alwaysContinueExecutionContext) OnDecisionMade(decisionLog *entity.TradingDecisionLog) error {
	return nil
}

func (alwaysContinueExecutionContext) ShouldContinue() bool {
	return true
}

func TestStartTradingBotUseCase_RunStrategyLoop_RecoversPanicAndStopsBot(t *testing.T) {
	tradingBotRepo := repository.NewTradeBotRepositoryInMemory()
	decisionLogRepo := repository.NewTradingDecisionLogRepositoryInMemory()
	useCase := NewStartTradingBotUseCaseWithServices(tradingBotRepo, decisionLogRepo, external.NewBinanceClientFake(), panickingMarketDataSource{}, alwaysContinueExecutionContext{})
	broker := &recordingLifecycleBroker{}
	useCase.messageBroker = broker
	useCase.exchangeName = "test-exchange"

	bot := saveRunningTestBot(t, tradingBotRepo)

	// Runs synchronously: an unrecovered panic here would crash the test binary.
	useCase.runStrategyLoop(bot)

	stored, _ := tradingBotRepo.GetTradeByID(bot.Id.GetValue())
	if stored.GetStatus() != entity.StatusStopped {
		t.Errorf("expected bot to be %s after panic, got %s", entity.StatusStopped, stored.GetStatus())
	}

	running, _ := tradingBotRepo.GetTradingBotsByStatus(entity.StatusRunning)
	if len(running) != 0 {
		t.Errorf("expected no RUNNING bots for auto-recovery, got %d", len(running))
	}

	msgs, _ := broker.messagesWithKey("trading_bot.stopped")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 trading_bot.stopped event, got %d", len(msgs))
	}
	assertLifecyclePayload(t, msgs[0], bot)
}

func TestStartTradingBotUseCase_RunStrategyLoop_RecoversPanicWithNilRepositories(t *testing.T) {
	useCase := NewStartTradingBotUseCaseWithServices(nil, nil, nil, panickingMarketDataSource{}, alwaysContinueExecutionContext{})

	bot := createTestTradingBot()
	_ = bot.Start()

	// The nil repository makes the loop panic; the recovery must not panic again.
	useCase.runStrategyLoop(bot)
}

// Note: Order placement tests moved to LiveTradingExecutionContext tests
// since order logic is now centralized there with LOT_SIZE validation

// Benchmark test for strategy execution
func BenchmarkStartTradingBotUseCase_StrategyExecution(b *testing.B) {
	useCase, tradingBotRepo, _, binanceClient := setupStartTradingBotUseCase()

	// Setup test data
	strongTrendKlines := external.CreateStrongTrendKlines()
	binanceClient.SetPredefinedKlines(strongTrendKlines)

	bot := createTestTradingBot()
	_ = tradingBotRepo.Save(bot)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		klines, _ := useCase.getMarketData(bot.GetSymbol().GetValue(), 300)
		strategy := bot.GetStrategy()
		_ = strategy.Decide(klines, bot)
	}
}
