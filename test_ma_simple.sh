#!/bin/bash

# Script simplificado sem dependência do jq

API_URL="http://localhost:8080/api/v1/trading/backtest"
AUTH_TOKEN="Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImphbG1laWRhY25AZ21haWwuY29tIiwiaXNzIjoiY3J5cGdvLW1hY2hpbmUiLCJzdWIiOiJqYWxtZWlkYWNuQGdtYWlsLmNvbSIsImV4cCI6MTc1MzEwMzE0MywibmJmIjoxNzUzMDE2NzQzLCJpYXQiOjE3NTMwMTY3NDN9.xlR_dt0oPjHwQBpH0NkUpPzwrgDUhqmG9xgxgB8iTwQ"

echo "🚀 TESTE SIMPLIFICADO DE PARÂMETROS - MOVING AVERAGE"
echo ""

# Função para extrair valor JSON sem jq
extract_json_value() {
    local json=$1
    local key=$2
    echo "$json" | grep -o "\"$key\":[^,}]*" | cut -d':' -f2 | tr -d ' "'
}

# Função para executar teste
run_test() {
    local fast=$1
    local slow=$2
    local stoploss=$3
    local profit=$4
    local desc=$5
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔍 $desc"
    echo "⚙️  Fast: $fast | Slow: $slow | SL: $stoploss% | Min: $profit%"
    echo ""
    
    # Executa curl e salva resposta
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
    
    # Verifica se teve resposta
    if [ -z "$RESPONSE" ]; then
        echo "❌ ERRO: Sem resposta da API"
        return
    fi
    
    # Verifica se é erro
    if echo "$RESPONSE" | grep -q "error"; then
        echo "❌ ERRO NA API:"
        echo "$RESPONSE"
        return
    fi
    
    # Extrai valores usando grep e sed
    ROI=$(echo "$RESPONSE" | grep -o '"roi":[^,}]*' | sed 's/"roi"://' | tr -d ' ')
    TRADES=$(echo "$RESPONSE" | grep -o '"total_trades":[^,}]*' | sed 's/"total_trades"://' | tr -d ' ')
    WIN_RATE=$(echo "$RESPONSE" | grep -o '"win_rate":"[^"]*"' | sed 's/"win_rate":"//' | tr -d '"')
    DRAWDOWN=$(echo "$RESPONSE" | grep -o '"max_drawdown":[^,}]*' | sed 's/"max_drawdown"://' | tr -d ' ')
    FINAL=$(echo "$RESPONSE" | grep -o '"final_capital":[^,}]*' | sed 's/"final_capital"://' | tr -d ' ')
    
    # Exibe resultados
    echo "📊 RESULTADOS:"
    echo "   💰 Capital Final: R$ ${FINAL:-???}"
    echo "   📈 ROI: ${ROI:-???}%"
    echo "   🎯 Win Rate: ${WIN_RATE:-???}"
    echo "   📉 Max Drawdown: ${DRAWDOWN:-???}%"
    echo "   🔄 Total Trades: ${TRADES:-???}"
    echo ""
}

# Executa testes
run_test 3 40 10 5 "CONFIGURAÇÃO ORIGINAL"
run_test 5 20 5 2 "DAY TRADING (5/20)"
run_test 7 25 5 3 "BALANCEADO (7/25)"
run_test 3 10 0 1 "SCALPING SEM STOPLOSS"
run_test 10 30 7 3 "SWING TRADING (10/30)"

echo "✅ Testes concluídos!"