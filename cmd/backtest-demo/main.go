package main

import (
	"crypgo-machine/src/application/usecase"
	"crypgo-machine/src/infra/external"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	fmt.Println("🚀 Executando Demo do Backtest com dados simulados...")

	// Create a fake client with predefined data
	client := external.NewBinanceClientFake()
	
	// Configure with whipsaw data to generate some trades
	client.SetPredefinedKlines(external.CreateWhipsawKlines())

	// Create backtest use case
	useCase := usecase.NewBacktestTradingBotUseCase(client)

	// Prepare input for 7-day backtest
	startDate := time.Now().AddDate(0, 0, -7) // 7 days ago
	endDate := time.Now()

	input := usecase.BacktestTradingBotInput{
		Symbol:   "BTCBRL",
		Strategy: "MovingAverage",
		StrategyParams: map[string]interface{}{
			"FastWindow": 3.0,
			"SlowWindow": 5.0,
		},
		StartDate:              startDate,
		EndDate:                endDate,
		InitialCapital:         1000.0,
		TradeAmount:            100.0,
		TradingFees:            0.1,
		MinimumProfitThreshold: 2.0, // 2% minimum profit
		Interval:               "1h",
	}

	// Print configuration
	fmt.Printf("📊 Configuração do Backtest:\n")
	fmt.Printf("   Symbol: %s\n", input.Symbol)
	fmt.Printf("   Strategy: %s (Fast: %.0f, Slow: %.0f)\n", input.Strategy, 
		input.StrategyParams["FastWindow"], input.StrategyParams["SlowWindow"])
	fmt.Printf("   Período: %s a %s\n", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	fmt.Printf("   Capital Inicial: %.2f BRL\n", input.InitialCapital)
	fmt.Printf("   Valor por Trade: %.2f BRL\n", input.TradeAmount)
	fmt.Printf("   Taxa de Trading: %.2f%%\n", input.TradingFees)
	fmt.Printf("   Threshold de Lucro Mínimo: %.2f%%\n", input.MinimumProfitThreshold)
	fmt.Printf("\n")

	// Execute backtest
	result, err := useCase.Execute(input)
	if err != nil {
		log.Fatalf("❌ Backtest falhou: %v", err)
	}

	// Print detailed results
	fmt.Printf("\n💾 Salvando resultados detalhados em 'demo_results.json'...\n")
	
	file, err := os.Create("demo_results.json")
	if err != nil {
		log.Printf("⚠️ Aviso: Falha ao criar arquivo de saída: %v", err)
	} else {
		defer file.Close()
		
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Printf("⚠️ Aviso: Falha ao escrever resultados: %v", err)
		} else {
			fmt.Printf("✅ Resultados salvos com sucesso!\n")
		}
	}

	// Print analysis of trades
	if len(result.Trades) > 0 {
		fmt.Printf("\n📊 ANÁLISE DETALHADA DOS TRADES:\n")
		fmt.Printf("   Trade#  | Entrada   | Saída     |    P&L    |   P&L%%   | Duração\n")
		fmt.Printf("   --------|-----------|-----------|-----------|---------|----------\n")
		
		for i, trade := range result.Trades {
			duration := trade.ExitTime.Sub(trade.EntryTime)
			status := "✅"
			if trade.PnL < 0 {
				status = "❌"
			}
			fmt.Printf("   %s %6d | %9.2f | %9.2f | %9.2f | %7.2f%% | %8s\n",
				status, i+1, trade.EntryPrice, trade.ExitPrice, trade.PnL, trade.PnLPercentage,
				duration.Truncate(time.Hour).String())
		}

		// Calculate some additional stats
		positivePnL := 0.0
		negativePnL := 0.0
		for _, trade := range result.Trades {
			if trade.PnL > 0 {
				positivePnL += trade.PnL
			} else {
				negativePnL += trade.PnL
			}
		}

		fmt.Printf("\n📈 ESTATÍSTICAS ADICIONAIS:\n")
		fmt.Printf("   💰 Lucro Total: %.2f BRL\n", positivePnL)
		fmt.Printf("   📉 Prejuízo Total: %.2f BRL\n", negativePnL)
		fmt.Printf("   ⚖️ Lucro Médio por Trade: %.2f BRL\n", result.TotalPnL/float64(result.TotalTrades))
		if result.TotalTrades > 0 {
			avgDuration := time.Duration(0)
			for _, trade := range result.Trades {
				avgDuration += trade.ExitTime.Sub(trade.EntryTime)
			}
			avgDuration = avgDuration / time.Duration(len(result.Trades))
			fmt.Printf("   ⏱️ Duração Média por Trade: %s\n", avgDuration.Truncate(time.Hour).String())
		}
	}

	fmt.Printf("\n✅ Demo do backtest concluído com sucesso!\n")
	fmt.Printf("🎯 O sistema está usando exatamente a mesma lógica do trading real.\n")
	fmt.Printf("💡 Teste com diferentes valores de minimum profit threshold para ver o efeito!\n")
}