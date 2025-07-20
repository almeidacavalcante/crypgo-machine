#!/bin/bash

# Script para comparar resultados COM e SEM stoploss

API_URL="http://localhost:8080/api/v1/trading/backtest"
AUTH_TOKEN="Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImphbG1laWRhY25AZ21haWwuY29tIiwiaXNzIjoiY3J5cGdvLW1hY2hpbmUiLCJzdWIiOiJqYWxtZWlkYWNuQGdtYWlsLmNvbSIsImV4cCI6MTc1MzEwMzE0MywibmJmIjoxNzUzMDE2NzQzLCJpYXQiOjE3NTMwMTY3NDN9.xlR_dt0oPjHwQBpH0NkUpPzwrgDUhqmG9xgxgB8iTwQ"

echo "🔬 COMPARAÇÃO: IMPACTO DO STOPLOSS NO MOVING AVERAGE"
echo "📅 Período: 01/04/2025 a 20/07/2025 | Par: SOLBRL"
echo ""

# Função para executar e comparar
compare_stoploss() {
    local fast=$1
    local slow=$2
    local profit=$3
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "⚙️ CONFIGURAÇÃO: Fast=$fast, Slow=$slow, MinProfit=$profit%"
    echo ""
    
    # Array de stoplosses para testar
    STOPLOSSES=(0 3 5 7 10 15)
    
    # Arquivo temporário para resultados
    TEMP_FILE="/tmp/stoploss_results_${fast}_${slow}_${profit}.txt"
    > "$TEMP_FILE"
    
    for sl in "${STOPLOSSES[@]}"; do
        if [ $sl -eq 0 ]; then
            echo -n "🚫 SEM Stoploss      : "
        else
            printf "📉 Stoploss %2d%%      : " $sl
        fi
        
        # Executa backtest
        RESPONSE=$(curl -s -X POST "$API_URL" \
            -H "Authorization: $AUTH_TOKEN" \
            -H "Content-Type: application/json" \
            -d "{
                \"strategy_name\": \"MovingAverage\",
                \"symbol\": \"SOLBRL\",
                \"params\": {
                    \"FastWindow\": $fast,
                    \"SlowWindow\": $slow,
                    \"StoplossThreshold\": $sl
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
        
        # Extrai métricas
        ROI=$(echo "$RESPONSE" | jq -r '.roi' 2>/dev/null || echo "0")
        TRADES=$(echo "$RESPONSE" | jq -r '.total_trades' 2>/dev/null || echo "0")
        WIN_RATE=$(echo "$RESPONSE" | jq -r '.win_rate' 2>/dev/null || echo "0%")
        DRAWDOWN=$(echo "$RESPONSE" | jq -r '.max_drawdown' 2>/dev/null || echo "0")
        FINAL=$(echo "$RESPONSE" | jq -r '.final_capital' 2>/dev/null || echo "10000")
        
        # Formata e exibe
        printf "ROI: %6.2f%% | Trades: %3s | Win: %6s | DD: %5.1f%% | Final: R$ %8.2f\n" \
            $ROI $TRADES "$WIN_RATE" $DRAWDOWN $FINAL
        
        # Salva para análise
        echo "$sl,$ROI,$TRADES,$WIN_RATE,$DRAWDOWN,$FINAL" >> "$TEMP_FILE"
    done
    
    echo ""
    echo "📊 ANÁLISE:"
    
    # Melhor ROI
    BEST_SL=$(sort -t',' -k2 -nr "$TEMP_FILE" | head -1 | cut -d',' -f1)
    BEST_ROI=$(sort -t',' -k2 -nr "$TEMP_FILE" | head -1 | cut -d',' -f2)
    
    if [ "$BEST_SL" -eq 0 ]; then
        echo "   🏆 Melhor resultado: SEM stoploss (ROI: ${BEST_ROI}%)"
    else
        echo "   🏆 Melhor resultado: Stoploss ${BEST_SL}% (ROI: ${BEST_ROI}%)"
    fi
    
    # Comparação sem stoploss vs melhor stoploss
    NO_SL_ROI=$(grep "^0," "$TEMP_FILE" | cut -d',' -f2)
    if [ "$BEST_SL" -ne 0 ]; then
        DIFF=$(echo "scale=2; $BEST_ROI - $NO_SL_ROI" | bc)
        echo "   📈 Ganho com stoploss: ${DIFF}% a mais que sem stoploss"
    fi
    
    rm -f "$TEMP_FILE"
}

# TESTE 1: Configuração original
compare_stoploss 3 40 5

echo ""

# TESTE 2: Day trading
compare_stoploss 5 20 2

echo ""

# TESTE 3: Swing trading
compare_stoploss 10 30 3

echo ""

# TESTE 4: Scalping
compare_stoploss 3 10 1

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "💡 CONCLUSÕES:"
echo "   • Stoploss pode proteger capital em mercados voláteis"
echo "   • Nem sempre stoploss melhora o ROI (depende da tendência)"
echo "   • Stoploss muito apertado (3-5%) pode sair prematuramente"
echo "   • Analise o número de trades: mais trades = mais oportunidades"
echo ""