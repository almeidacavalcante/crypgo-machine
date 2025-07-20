#!/bin/bash

# Script para comparar resultados de diferentes configurações Moving Average

API_URL="http://localhost:8080/api/v1/trading/backtest"
AUTH_TOKEN="Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImphbG1laWRhY25AZ21haWwuY29tIiwiaXNzIjoiY3J5cGdvLW1hY2hpbmUiLCJzdWIiOiJqYWxtZWlkYWNuQGdtYWlsLmNvbSIsImV4cCI6MTc1MzEwMzE0MywibmJmIjoxNzUzMDE2NzQzLCJpYXQiOjE3NTMwMTY3NDN9.xlR_dt0oPjHwQBpH0NkUpPzwrgDUhqmG9xgxgB8iTwQ"

# Arquivo temporário para armazenar resultados
RESULTS_FILE="/tmp/ma_results_$(date +%s).csv"
echo "Config,FastWindow,SlowWindow,Stoploss,MinProfit,FinalCapital,ROI,WinRate,TotalTrades,MaxDrawdown" > "$RESULTS_FILE"

# Função para executar backtest
run_backtest() {
    local fast=$1
    local slow=$2
    local stoploss=$3
    local profit=$4
    local description=$5
    
    # Executa o curl
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
    
    # Extrai valores usando jq
    if command -v jq &> /dev/null; then
        FINAL_CAPITAL=$(echo "$RESPONSE" | jq -r '.data.final_capital // 0')
        ROI=$(echo "$RESPONSE" | jq -r '.data.roi_percentage // 0')
        WIN_RATE=$(echo "$RESPONSE" | jq -r '.data.win_rate // "0%"' | tr -d '%')
        TOTAL_TRADES=$(echo "$RESPONSE" | jq -r '.data.total_trades // 0')
        MAX_DRAWDOWN=$(echo "$RESPONSE" | jq -r '.data.max_drawdown // "0%"' | grep -o '[0-9.]*' | head -1)
        
        # Salva no arquivo
        echo "$description,$fast,$slow,$stoploss,$profit,$FINAL_CAPITAL,$ROI,$WIN_RATE,$TOTAL_TRADES,$MAX_DRAWDOWN" >> "$RESULTS_FILE"
        
        # Exibe resultado formatado
        printf "%-25s | ROI: %7.2f%% | Trades: %2d | Win: %5s%% | DD: %5s%% | Final: R$ %8.2f\n" \
            "$description" "$ROI" "$TOTAL_TRADES" "$WIN_RATE" "$MAX_DRAWDOWN" "$FINAL_CAPITAL"
    fi
}

echo "🚀 COMPARAÇÃO DE CONFIGURAÇÕES MOVING AVERAGE"
echo "📅 Período: 01/04/2025 a 20/07/2025 | Par: SOLBRL"
echo ""
echo "Configuração              | ROI         | Trades | Win Rate | Drawdown | Capital Final"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Executa os testes
run_backtest 3 40 10 5 "Original (3/40)"
run_backtest 3 10 0 1 "Scalping sem SL"
run_backtest 3 10 3 1 "Scalping SL 3%"
run_backtest 5 20 5 2 "Day Trading (5/20)"
run_backtest 10 30 7 3 "Swing Trading (10/30)"
run_backtest 7 50 10 5 "Conservador (7/50)"
run_backtest 5 25 0 7 "Sem SL, Profit 7%"
run_backtest 7 25 5 3 "Balanceado (7/25)"
run_backtest 12 40 15 7 "Ultra Conserv (12/40)"
run_backtest 5 15 7 2 "Cripto Volátil (5/15)"

# Análises extras
run_backtest 3 20 5 2 "Fast Scalp (3/20)"
run_backtest 7 30 0 3 "Médio sem SL (7/30)"
run_backtest 10 40 10 5 "Longo prazo (10/40)"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Análise dos resultados
echo "📊 ANÁLISE DOS RESULTADOS:"
echo ""

# Melhor ROI
echo "🏆 TOP 3 - MELHOR ROI:"
sort -t',' -k6 -nr "$RESULTS_FILE" | head -4 | tail -3 | while IFS=',' read -r config fast slow sl profit final roi winrate trades dd; do
    printf "   %s: ROI %.2f%% (Win: %s%%, Trades: %s)\n" "$config" "$roi" "$winrate" "$trades"
done

echo ""
echo "📈 TOP 3 - MAIS TRADES (Mais oportunidades):"
sort -t',' -k9 -nr "$RESULTS_FILE" | head -4 | tail -3 | while IFS=',' read -r config fast slow sl profit final roi winrate trades dd; do
    printf "   %s: %s trades (ROI: %.2f%%, Win: %s%%)\n" "$config" "$trades" "$roi" "$winrate"
done

echo ""
echo "🛡️ TOP 3 - MENOR DRAWDOWN (Mais seguro):"
sort -t',' -k10 -n "$RESULTS_FILE" | head -4 | tail -3 | while IFS=',' read -r config fast slow sl profit final roi winrate trades dd; do
    printf "   %s: DD %.2f%% (ROI: %.2f%%, Trades: %s)\n" "$config" "$dd" "$roi" "$trades"
done

echo ""
echo "💡 INSIGHTS:"

# Análise de Stoploss
AVG_WITH_SL=$(awk -F',' '$4>0 {sum+=$6; count++} END {if(count>0) print sum/count; else print 0}' "$RESULTS_FILE")
AVG_WITHOUT_SL=$(awk -F',' '$4==0 {sum+=$6; count++} END {if(count>0) print sum/count; else print 0}' "$RESULTS_FILE")

echo "   • Média ROI COM Stoploss: $(printf "%.2f%%" $AVG_WITH_SL)"
echo "   • Média ROI SEM Stoploss: $(printf "%.2f%%" $AVG_WITHOUT_SL)"

# Melhor Fast Window
BEST_FAST=$(awk -F',' 'NR>1 {sum[$2]+=$6; count[$2]++} END {for(f in sum) print f, sum[f]/count[f]}' "$RESULTS_FILE" | sort -k2 -nr | head -1 | awk '{print $1}')
echo "   • Melhor Fast Window em média: $BEST_FAST"

# Melhor Slow Window
BEST_SLOW=$(awk -F',' 'NR>1 {sum[$3]+=$6; count[$3]++} END {for(s in sum) print s, sum[s]/count[s]}' "$RESULTS_FILE" | sort -k2 -nr | head -1 | awk '{print $1}')
echo "   • Melhor Slow Window em média: $BEST_SLOW"

echo ""
echo "📌 RECOMENDAÇÃO FINAL:"
# Pega a melhor configuração geral
BEST_CONFIG=$(sort -t',' -k6 -nr "$RESULTS_FILE" | head -2 | tail -1)
echo "   $BEST_CONFIG" | awk -F',' '{printf "   Melhor configuração: %s\n   Parâmetros: Fast=%s, Slow=%s, SL=%s%%, MinProfit=%s%%\n   Resultado: ROI=%.2f%%, WinRate=%s%%, Trades=%s\n", $1, $2, $3, $4, $5, $6, $7, $8}'

# Limpa arquivo temporário
rm -f "$RESULTS_FILE"