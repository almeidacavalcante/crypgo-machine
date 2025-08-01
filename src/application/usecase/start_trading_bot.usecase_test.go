package usecase

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"crypgo-machine/src/infra/repository"
	"testing"
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
