#!/bin/bash

# Script de debug para verificar a resposta da API

API_URL="http://localhost:8080/api/v1/trading/backtest"
AUTH_TOKEN="Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImphbG1laWRhY25AZ21haWwuY29tIiwiaXNzIjoiY3J5cGdvLW1hY2hpbmUiLCJzdWIiOiJqYWxtZWlkYWNuQGdtYWlsLmNvbSIsImV4cCI6MTc1MzEwMzE0MywibmJmIjoxNzUzMDE2NzQzLCJpYXQiOjE3NTMwMTY3NDN9.xlR_dt0oPjHwQBpH0NkUpPzwrgDUhqmG9xgxgB8iTwQ"

echo "🔍 TESTE DE DEBUG - VERIFICANDO RESPOSTA DA API"
echo ""

# Teste simples
echo "📡 Enviando request para API..."
echo ""

# Mostra o comando curl completo
echo "COMANDO CURL:"
echo "-------------"
cat << 'EOF'
curl -X POST "http://localhost:8080/api/v1/trading/backtest" \
    -H "Authorization: Bearer TOKEN..." \
    -H "Content-Type: application/json" \
    -d '{
        "strategy_name": "MovingAverage",
        "symbol": "SOLBRL",
        "params": {
            "FastWindow": 3,
            "SlowWindow": 40,
            "StoplossThreshold": 10
        },
        "start_date": "2025-04-01T00:00:00Z",
        "end_date": "2025-07-20T00:00:00Z",
        "interval": "5m",
        "initial_capital": 10000.0,
        "trade_amount": 1000.0,
        "currency": "BRL",
        "trading_fees": 0.1,
        "minimum_profit_threshold": 5,
        "use_binance_data": true
    }'
EOF

echo ""
echo "RESPOSTA RAW:"
echo "-------------"

# Executa e mostra resposta completa
RESPONSE=$(curl -s -X POST "$API_URL" \
    -H "Authorization: $AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "strategy_name": "MovingAverage",
        "symbol": "SOLBRL",
        "params": {
            "FastWindow": 3,
            "SlowWindow": 40,
            "StoplossThreshold": 10
        },
        "start_date": "2025-04-01T00:00:00Z",
        "end_date": "2025-07-20T00:00:00Z",
        "interval": "5m",
        "initial_capital": 10000.0,
        "trade_amount": 1000.0,
        "currency": "BRL",
        "trading_fees": 0.1,
        "minimum_profit_threshold": 5,
        "use_binance_data": true
    }')

echo "$RESPONSE"

echo ""
echo "RESPOSTA FORMATADA (se for JSON válido):"
echo "-----------------------------------------"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "❌ Não foi possível formatar como JSON"

echo ""
echo "TESTE DE EXTRAÇÃO DE CAMPOS:"
echo "-----------------------------"
echo "ROI: $(echo "$RESPONSE" | jq -r '.roi' 2>/dev/null || echo "ERRO")"
echo "Total Trades: $(echo "$RESPONSE" | jq -r '.total_trades' 2>/dev/null || echo "ERRO")"
echo "Win Rate: $(echo "$RESPONSE" | jq -r '.win_rate' 2>/dev/null || echo "ERRO")"
echo "Final Capital: $(echo "$RESPONSE" | jq -r '.final_capital' 2>/dev/null || echo "ERRO")"

echo ""
echo "STATUS HTTP:"
echo "-------------"
# Testa com verbose para ver status HTTP
curl -s -o /dev/null -w "HTTP Status: %{http_code}\n" -X POST "$API_URL" \
    -H "Authorization: $AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "strategy_name": "MovingAverage",
        "symbol": "SOLBRL",
        "params": {
            "FastWindow": 3,
            "SlowWindow": 40,
            "StoplossThreshold": 10
        },
        "start_date": "2025-04-01T00:00:00Z",
        "end_date": "2025-07-20T00:00:00Z",
        "interval": "5m",
        "initial_capital": 10000.0,
        "trade_amount": 1000.0,
        "currency": "BRL",
        "trading_fees": 0.1,
        "minimum_profit_threshold": 5,
        "use_binance_data": true
    }'

echo ""
echo "✅ Debug concluído!"