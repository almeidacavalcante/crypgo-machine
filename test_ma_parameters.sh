#!/bin/bash

# Script para testar diferentes parâmetros de Moving Average
# Mantém vela de 5m e varia FastWindow, SlowWindow, StoplossThreshold e MinimumProfitThreshold

API_URL="http://localhost:8080/api/v1/trading/backtest"
AUTH_TOKEN="Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImphbG1laWRhY25AZ21haWwuY29tIiwiaXNzIjoiY3J5cGdvLW1hY2hpbmUiLCJzdWIiOiJqYWxtZWlkYWNuQGdtYWlsLmNvbSIsImV4cCI6MTc1MzEwMzE0MywibmJmIjoxNzUzMDE2NzQzLCJpYXQiOjE3NTMwMTY3NDN9.xlR_dt0oPjHwQBpH0NkUpPzwrgDUhqmG9xgxgB8iTwQ"

# Arrays com diferentes valores para testar
FAST_WINDOWS=(3 5 7 10 12)
SLOW_WINDOWS=(20 25 30 40 50)
STOPLOSS_THRESHOLDS=(0 3 5 7 10)
PROFIT_THRESHOLDS=(1 2 3 5 7)

# Arquivo para salvar resultados
RESULTS_FILE="ma_backtest_results_$(date +%Y%m%d_%H%M%S).csv"

# Cabeçalho do CSV
echo "FastWindow,SlowWindow,Stoploss,MinProfit,TotalTrades,WinRate,ROI,MaxDrawdown,FinalCapital" > "$RESULTS_FILE"

echo "🚀 Iniciando testes de parâmetros Moving Average..."
echo "📊 Resultados serão salvos em: $RESULTS_FILE"
echo ""

# Contador de testes
TEST_COUNT=0
TOTAL_TESTS=$((${#FAST_WINDOWS[@]} * ${#SLOW_WINDOWS[@]} * ${#STOPLOSS_THRESHOLDS[@]} * ${#PROFIT_THRESHOLDS[@]}))

# Loop através de todas as combinações
for fast in "${FAST_WINDOWS[@]}"; do
    for slow in "${SLOW_WINDOWS[@]}"; do
        # Apenas testa se slow > fast
        if [ $slow -le $fast ]; then
            continue
        fi
        
        for stoploss in "${STOPLOSS_THRESHOLDS[@]}"; do
            for profit in "${PROFIT_THRESHOLDS[@]}"; do
                TEST_COUNT=$((TEST_COUNT + 1))
                
                echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                echo "📈 Teste $TEST_COUNT/$TOTAL_TESTS"
                echo "⚙️  Fast: $fast | Slow: $slow | SL: $stoploss% | Min Profit: $profit%"
                
                # Executa o backtest
                RESPONSE=$(curl -s -X POST "$API_URL" \
                    -H "Authorization: $AUTH_TOKEN" \
                    -H "Content-Type: application/json" \
                    -d "{
                        \"strategy_name\": \"MovingAverage\",
                        \"symbol\": \"SOLBRL\",
                        \"params\": {
                            \"FastWindow\": $fast,
                            \"SlowWindow\": $slow,
                            \"StoplossThreshold\": $stoploss
                        },
                        \"start_date\": \"2025-04-01T00:00:00Z\",
                        \"end_date\": \"2025-07-20T00:00:00Z\",
                        \"interval\": \"5m\",
                        \"initial_capital\": 10000.0,
                        \"trade_amount\": 1000.0,
                        \"currency\": \"BRL\",
                        \"trading_fees\": 0.1,
                        \"minimum_profit_threshold\": $profit,
                        \"use_binance_data\": true
                    }")
                
                # Extrai métricas do JSON response
                TOTAL_TRADES=$(echo "$RESPONSE" | grep -o '"total_trades":[0-9]*' | grep -o '[0-9]*' || echo "0")
                WIN_RATE=$(echo "$RESPONSE" | grep -o '"win_rate":"[^"]*"' | cut -d'"' -f4 || echo "0%")
                ROI=$(echo "$RESPONSE" | grep -o '"roi":[0-9.-]*' | grep -o '[0-9.-]*' || echo "0")
                MAX_DRAWDOWN=$(echo "$RESPONSE" | grep -o '"max_drawdown":[0-9.-]*' | grep -o '[0-9.-]*' || echo "0")
                FINAL_CAPITAL=$(echo "$RESPONSE" | grep -o '"final_capital":[0-9.-]*' | grep -o '[0-9.-]*' || echo "10000")
                
                # Remove % do win_rate se existir
                WIN_RATE=${WIN_RATE%\%}
                
                # Mostra resumo
                echo "📊 Resultados:"
                echo "   • Trades: $TOTAL_TRADES"
                echo "   • Win Rate: $WIN_RATE%"
                echo "   • ROI: $ROI%"
                echo "   • Max Drawdown: $MAX_DRAWDOWN%"
                echo "   • Capital Final: R$ $FINAL_CAPITAL"
                
                # Salva no CSV
                echo "$fast,$slow,$stoploss,$profit,$TOTAL_TRADES,$WIN_RATE,$ROI,$MAX_DRAWDOWN,$FINAL_CAPITAL" >> "$RESULTS_FILE"
                
                # Pequena pausa para não sobrecarregar o servidor
                sleep 1
            done
        done
    done
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Testes concluídos!"
echo "📊 Resultados salvos em: $RESULTS_FILE"
echo ""

# Mostra os 10 melhores resultados por ROI
echo "🏆 TOP 10 MELHORES RESULTADOS (por ROI):"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Fast | Slow | SL% | MinP% | Trades | WinRate | ROI% | Drawdown | Final"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
tail -n +2 "$RESULTS_FILE" | sort -t',' -k7 -nr | head -10 | while IFS=',' read -r fast slow sl profit trades winrate roi drawdown final; do
    printf "%-4s | %-4s | %-3s | %-5s | %-6s | %-7s | %-5s | %-8s | %.0f\n" \
        "$fast" "$slow" "$sl" "$profit" "$trades" "$winrate%" "$roi" "$drawdown%" "$final"
done

echo ""
echo "🎯 ANÁLISE RÁPIDA:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Análise de melhor stoploss
echo -n "📉 Melhor Stoploss: "
tail -n +2 "$RESULTS_FILE" | awk -F',' '{sum[$3]+=$7; count[$3]++} END {for (sl in sum) print sl, sum[sl]/count[sl]}' | sort -k2 -nr | head -1 | awk '{print $1"%"}'

# Análise de melhor profit threshold
echo -n "💰 Melhor Min Profit: "
tail -n +2 "$RESULTS_FILE" | awk -F',' '{sum[$4]+=$7; count[$4]++} END {for (p in sum) print p, sum[p]/count[p]}' | sort -k2 -nr | head -1 | awk '{print $1"%"}'

# Análise de melhor fast window
echo -n "⚡ Melhor Fast Window: "
tail -n +2 "$RESULTS_FILE" | awk -F',' '{sum[$1]+=$7; count[$1]++} END {for (f in sum) print f, sum[f]/count[f]}' | sort -k2 -nr | head -1 | awk '{print $1}'

# Análise de melhor slow window
echo -n "🐌 Melhor Slow Window: "
tail -n +2 "$RESULTS_FILE" | awk -F',' '{sum[$2]+=$7; count[$2]++} END {for (s in sum) print s, sum[s]/count[s]}' | sort -k2 -nr | head -1 | awk '{print $1}'