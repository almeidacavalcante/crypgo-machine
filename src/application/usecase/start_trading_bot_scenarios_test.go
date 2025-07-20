package usecase

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/external"
	"testing"

	"github.com/adshao/go-binance/v2"
)

func TestMovingAverageStrategy_AntiWhipsawScenarios(t *testing.T) {
	tests := []struct {
		name           string
		klines         []*binance.Kline
		minimumSpread  float64
		expectedDecision entity.TradingDecision
		expectedReason   string
		botPositioned    bool
	}{
		{
			name:             "LongUptrend_WithSpread_ShouldHold",
			klines:           external.CreateWhipsawKlines(),
			minimumSpread:    0.1,
			expectedDecision: entity.Hold,
			expectedReason:   "fast_above_slow_wait_for_dip",
			botPositioned:    false,
		},
		{
			name:             "StrongTrend_LargeSpread_ShouldHold",
			klines:           external.CreateStrongTrendKlines(),
			minimumSpread:    0.1,
			expectedDecision: entity.Hold,
			expectedReason:   "fast_above_slow_wait_for_dip",
			botPositioned:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			useCase, tradingBotRepo, _, binanceClient := setupStartTradingBotUseCase()
			binanceClient.SetPredefinedKlines(tt.klines)
			
			// Create bot with custom minimum spread
			symbol, _ := vo.NewSymbol("BTCUSDT")
			minimumSpread, _ := vo.NewMinimumSpread(tt.minimumSpread)
			strategy := entity.NewMovingAverageStrategyWithSpread(7, 40, minimumSpread)
			bot := entity.NewTradingBot(symbol, 0.001, strategy, 60, 10000.0, 1000.0, "USDT", 0.001, 0.0, true)
			
			// Set bot position if needed
			if tt.botPositioned {
				_ = bot.GetIntoPosition()
			}
			
			err := tradingBotRepo.Save(bot)
			if err != nil {
				t.Fatalf("Failed to save bot: %v", err)
			}
			
			// Execute strategy
			klines, err := useCase.getMarketData(bot.GetSymbol().GetValue(), 300)
			if err != nil {
				t.Fatalf("Failed to get market data: %v", err)
			}
			
			analysisResult := strategy.Decide(klines, bot)
			
			// Verify decision
			if analysisResult.Decision != tt.expectedDecision {
				t.Errorf("Expected decision %v, got %v", tt.expectedDecision, analysisResult.Decision)
			}
			
			// Verify reason
			reason, ok := analysisResult.AnalysisData["reason"].(string)
			if !ok {
				t.Fatal("Expected reason in analysis data")
			}
			
			if reason != tt.expectedReason {
				t.Errorf("Expected reason '%s', got '%s'", tt.expectedReason, reason)
			}
			
			// Verify spread calculation
			hasSufficientSpread, ok := analysisResult.AnalysisData["hasSufficientSpread"].(bool)
			if !ok {
				t.Fatal("Expected hasSufficientSpread in analysis data")
			}
			
			// For decisions other than Hold (when conditions are met), spread should be sufficient
			if tt.expectedDecision != entity.Hold {
				if !hasSufficientSpread {
					t.Error("Expected sufficient spread for non-Hold decision")
				}
			}
		})
	}
}

func TestTradingDecisionLog_FullWorkflow(t *testing.T) {
	useCase, tradingBotRepo, decisionLogRepo, binanceClient := setupStartTradingBotUseCase()
	
	// Setup scenario with multiple decisions
	binanceClient.SetPredefinedKlines(external.CreateStrongTrendKlines())
	
	// Create bot
	bot := createTestTradingBot()
	err := tradingBotRepo.Save(bot)
	if err != nil {
		t.Fatalf("Failed to save bot: %v", err)
	}
	
	// Simulate multiple strategy executions
	for i := 0; i < 3; i++ {
		klines, err := useCase.getMarketData(bot.GetSymbol().GetValue(), 300)
		if err != nil {
			t.Fatalf("Failed to get market data: %v", err)
		}
		
		strategy := bot.GetStrategy()
		analysisResult := strategy.Decide(klines, bot)
		
		// Create decision log
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
		
		// If it's a buy decision, position the bot for next iteration
		if analysisResult.Decision == entity.Buy {
			_ = bot.GetIntoPosition()
		} else if analysisResult.Decision == entity.Sell {
			_ = bot.GetOutOfPosition()
		}
	}
	
	// Verify all logs were saved
	logs, err := decisionLogRepo.GetByTradingBotId(bot.Id.GetValue())
	if err != nil {
		t.Fatalf("Failed to get decision logs: %v", err)
	}
	
	if len(logs) != 3 {
		t.Errorf("Expected 3 decision logs, got %d", len(logs))
	}
	
	// Verify logs are ordered by timestamp (most recent first)
	for i := 1; i < len(logs); i++ {
		if logs[i-1].GetTimestamp().Before(logs[i].GetTimestamp()) {
			t.Error("Expected logs to be ordered by timestamp DESC")
		}
	}
	
	// Test limit functionality
	limitedLogs, err := decisionLogRepo.GetByTradingBotIdWithLimit(bot.Id.GetValue(), 2)
	if err != nil {
		t.Fatalf("Failed to get limited decision logs: %v", err)
	}
	
	if len(limitedLogs) != 2 {
		t.Errorf("Expected 2 limited logs, got %d", len(limitedLogs))
	}
}

func TestMarketDataConversion(t *testing.T) {
	useCase, _, _, binanceClient := setupStartTradingBotUseCase()
	
	// Setup test klines
	testKlines := []*binance.Kline{
		{
			Open:      "100.50",
			Close:     "101.75",
			High:      "102.00",
			Low:       "100.25",
			Volume:    "1000.0",
			CloseTime: 1640995200000,
		},
	}
	binanceClient.SetPredefinedKlines(testKlines)
	
	// Convert to domain klines
	domainKlines, err := useCase.getMarketData("BTCUSDT", 300)
	if err != nil {
		t.Fatalf("Failed to convert market data: %v", err)
	}
	
	if len(domainKlines) != 1 {
		t.Fatalf("Expected 1 kline, got %d", len(domainKlines))
	}
	
	kline := domainKlines[0]
	
	// Verify conversion
	if kline.Open() != 100.50 {
		t.Errorf("Expected open 100.50, got %f", kline.Open())
	}
	
	if kline.Close() != 101.75 {
		t.Errorf("Expected close 101.75, got %f", kline.Close())
	}
	
	if kline.High() != 102.00 {
		t.Errorf("Expected high 102.00, got %f", kline.High())
	}
	
	if kline.Low() != 100.25 {
		t.Errorf("Expected low 100.25, got %f", kline.Low())
	}
	
	if kline.Volume() != 1000.0 {
		t.Errorf("Expected volume 1000.0, got %f", kline.Volume())
	}
	
	if kline.CloseTime() != 1640995200000 {
		t.Errorf("Expected close time 1640995200000, got %d", kline.CloseTime())
	}
}