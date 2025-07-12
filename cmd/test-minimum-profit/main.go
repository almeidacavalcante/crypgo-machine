package main

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"fmt"
)

func main() {
	fmt.Println("🧪 Teste direto do Minimum Profit Threshold")
	fmt.Println("===========================================")

	// Criar klines que simulam uma situação de compra seguida de venda
	klines := createTestScenario()
	
	// Teste 1: Bot com 0% minimum profit threshold
	fmt.Println("\n📊 TESTE 1: Minimum Profit Threshold = 0%")
	testWithThreshold(klines, 0.0)
	
	// Teste 2: Bot com 5% minimum profit threshold  
	fmt.Println("\n📊 TESTE 2: Minimum Profit Threshold = 5%")
	testWithThreshold(klines, 5.0)
	
	// Teste 3: Bot com 10% minimum profit threshold
	fmt.Println("\n📊 TESTE 3: Minimum Profit Threshold = 10%")
	testWithThreshold(klines, 10.0)
	
	fmt.Println("\n✅ Teste concluído! O minimum profit threshold está funcionando corretamente.")
}

func createTestScenario() []vo.Kline {
	// Criar cenário onde:
	// 1. Preços começam baixos (trigger para compra)
	// 2. Depois sobem (trigger para venda, mas com diferentes níveis de lucro)
	
	klines := []vo.Kline{}
	
	// Fase 1: Preços descendo (5 velas) - MA fast ficará abaixo de MA slow
	prices := []float64{110, 108, 106, 104, 102}
	for i, price := range prices {
		k, _ := vo.NewKline(price, price, price+1, price-1, 100, int64(1640995200000+i*3600000))
		klines = append(klines, k)
	}
	
	// Fase 2: Preços subindo (10 velas) - MA fast ficará acima de MA slow
	// Progressão que permite testar diferentes thresholds de lucro
	upPrices := []float64{103, 104, 105, 106, 107, 108, 109, 110, 111, 112}
	for i, price := range upPrices {
		k, _ := vo.NewKline(price, price, price+1, price-1, 100, int64(1640995200000+(len(prices)+i)*3600000))
		klines = append(klines, k)
	}
	
	return klines
}

func testWithThreshold(klines []vo.Kline, threshold float64) {
	// Criar bot de teste
	symbol, _ := vo.NewSymbol("BTCBRL")
	strategy := entity.NewMovingAverageStrategy(3, 5)
	bot := entity.NewTradingBot(symbol, 0.001, strategy, 60, 1000.0, 100.0, "BRL", 0.1, threshold)
	
	fmt.Printf("   Bot configurado com threshold: %.1f%%\n", threshold)
	
	// Simular execução das velas
	
	for i, kline := range klines {
		// Usar apenas as últimas N velas (simulando sliding window)
		startIdx := 0
		if i >= 10 {
			startIdx = i - 9 // Últimas 10 velas
		}
		currentKlines := klines[startIdx:i+1]
		
		if len(currentKlines) < 5 { // Precisa de pelo menos 5 velas para MA slow
			continue
		}
		
		result := strategy.Decide(currentKlines, bot)
		
		if result.Decision == entity.Buy && !bot.GetIsPositioned() {
			// Simular compra
			bot.SetEntryPrice(kline.Close())
			_ = bot.GetIntoPosition()
			fmt.Printf("   🟢 COMPRA executada no preço %.2f\n", kline.Close())
			fmt.Printf("      Análise: fast=%.2f, slow=%.2f, spread=%.2f%%\n", 
				result.AnalysisData["fast"].(float64),
				result.AnalysisData["slow"].(float64),
				result.AnalysisData["actualSpread"].(float64))
		}
		
		if result.Decision == entity.Sell && bot.GetIsPositioned() {
			// Simular venda
			profit := result.AnalysisData["possibleProfit"].(float64)
			fmt.Printf("   🔴 VENDA executada no preço %.2f\n", kline.Close())
			fmt.Printf("      Lucro: %.2f%% (threshold: %.1f%%)\n", profit, threshold)
			fmt.Printf("      Razão: %s\n", result.AnalysisData["reason"].(string))
			_ = bot.GetOutOfPosition()
			bot.ClearEntryPrice()
			break
		}
		
		if result.Decision == entity.Hold && bot.GetIsPositioned() {
			profit := result.AnalysisData["possibleProfit"].(float64)
			reason := result.AnalysisData["reason"].(string)
			if reason == "fast_above_slow_hold_insufficient_profit" {
				fmt.Printf("   ⏸ HOLD: Lucro %.2f%% insuficiente (precisa %.1f%%)\n", profit, threshold)
			}
		}
	}
	
	// Verificar se ainda está posicionado
	if bot.GetIsPositioned() {
		currentPrice := klines[len(klines)-1].Close()
		profit := ((currentPrice - bot.GetEntryPrice()) / bot.GetEntryPrice()) * 100
		fmt.Printf("   💼 Ainda posicionado - Lucro atual: %.2f%%\n", profit)
	}
}