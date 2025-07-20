#!/bin/bash

# Script otimizado para testar as melhores combinações de parâmetros Moving Average
# Foca em combinações que historicamente funcionam bem

API_URL="http://localhost:8080/api/v1/trading/backtest"
AUTH_TOKEN="Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImphbG1laWRhY25AZ21haWwuY29tIiwiaXNzIjoiY3J5cGdvLW1hY2hpbmUiLCJzdWIiOiJqYWxtZWlkYWNuQGdtYWlsLmNvbSIsImV4cCI6MTc1MzEwMzE0MywibmJmIjoxNzUzMDE2NzQzLCJpYXQiOjE3NTMwMTY3NDN9.xlR_dt0oPjHwQBpH0NkUpPzwrgDUhqmG9xgxgB8iTwQ"

# Função para executar backtest
run_backtest() {
    local fast=$1
    local slow=$2
    local stoploss=$3
    local profit=$4
    local description=$5
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔍 $description"
    echo "⚙️  Fast: $fast | Slow: $slow | Stoploss: $stoploss% | Min Profit: $profit%"
    echo ""
    
    # Executa o curl e captura a resposta completa
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
            \"start_date\": \"2025-07-12T00:00:00Z\",
            \"end_date\": \"2025-07-20T00:00:00Z\",
            \"interval\": \"30m\",
            \"initial_capital\": 10000.0,
            \"trade_amount\": 1000.0,
            \"currency\": \"BRL\",
            \"trading_fees\": 0.1,
            \"minimum_profit_threshold\": $profit,
            \"use_binance_data\": true
        }")
    
    # Remove debug para saída mais limpa
    # echo "DEBUG - Resposta Raw:"
    # echo "$RESPONSE"
    # echo "---"
    
    # Verifica se há erro
    if echo "$RESPONSE" | grep -q "error"; then
        echo "❌ ERRO NA API:"
        echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
        return
    fi
    
    # Se a resposta estiver vazia
    if [ -z "$RESPONSE" ]; then
        echo "❌ ERRO: Resposta vazia da API"
        return
    fi
    
    # Tenta processar com jq
    if command -v jq &> /dev/null; then
        echo "$RESPONSE" | jq -r '
        "📊 RESULTADOS:",
        "   💰 Capital Inicial: R$ \(.data.initial_capital)",
        "   💵 Capital Final: R$ \(.data.final_capital)",
        "   📈 ROI: \(.data.roi_percentage)%",
        "   🎯 Win Rate: \(.data.win_rate)",
        "   📉 Max Drawdown: \(.data.max_drawdown)",
        "   🔄 Total de Trades: \(.data.total_trades)",
        "   ✅ Trades Vencedores: \(.data.winning_trades)",
        "   ❌ Trades Perdedores: \(.data.losing_trades)",
        "   💸 P&L Total: \(.data.total_profit_loss)"
    ' 2>/dev/null || echo "❌ Erro ao processar resposta com jq"
    else
        # Fallback sem jq
        echo "📊 RESULTADOS (sem jq):"
        echo "   💰 Capital Final: $(echo "$RESPONSE" | grep -o '"final_capital":[^,}]*' | cut -d':' -f2)"
        echo "   📈 ROI: $(echo "$RESPONSE" | grep -o '"roi_percentage":[^,}]*' | cut -d':' -f2)%"
        echo "   🎯 Win Rate: $(echo "$RESPONSE" | grep -o '"win_rate":"[^"]*"' | cut -d':' -f2)"
        echo "   🔄 Total Trades: $(echo "$RESPONSE" | grep -o '"total_trades":[^,}]*' | cut -d':' -f2)"
    fi
}

echo "🚀 TESTANDO MELHORES COMBINAÇÕES DE PARÂMETROS PARA MOVING AVERAGE"
echo "📅 Período: 01/04/2025 a 20/07/2025 | Intervalo: 5m | Par: SOLBRL"
echo ""

# TESTE 1: Configuração original do usuário
run_backtest 3 40 10 5 "CONFIGURAÇÃO ORIGINAL (Baseline)"

# TESTE 2: Scalping agressivo sem stoploss
run_backtest 3 10 0 1 "SCALPING AGRESSIVO (Fast: 3, Slow: 10, Sem SL)"

# TESTE 3: Scalping com stoploss apertado
run_backtest 3 10 3 1 "SCALPING COM STOPLOSS APERTADO (SL: 3%)"

# TESTE 4: Day trading clássico
run_backtest 5 20 5 2 "DAY TRADING CLÁSSICO (5/20, SL: 5%)"

# TESTE 5: Swing trading
run_backtest 10 30 7 3 "SWING TRADING (10/30, SL: 7%)"

# TESTE 6: Conservador
run_backtest 7 50 10 5 "CONSERVADOR (7/50, SL: 10%, Min: 5%)"

# TESTE 7: Sem stoploss, profit alto
run_backtest 5 25 0 7 "SEM STOPLOSS, PROFIT ALTO (5/25, Min: 7%)"

# TESTE 8: Balanceado
run_backtest 7 25 5 3 "BALANCEADO (7/25, SL: 5%, Min: 3%)"

# TESTE 9: Ultra conservador
run_backtest 12 40 15 7 "ULTRA CONSERVADOR (12/40, SL: 15%, Min: 7%)"

# TESTE 10: Otimizado para cripto volátil
run_backtest 5 15 7 2 "CRIPTO VOLÁTIL (5/15, SL: 7%, Min: 2%)"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ TESTES CONCLUÍDOS!"
echo ""
echo "💡 DICAS:"
echo "   • ROI positivo com poucos trades pode indicar overfitting"
echo "   • Win rate > 50% com muitos trades é um bom sinal"
echo "   • Max drawdown < 10% indica boa gestão de risco"
echo "   • Stoploss muito apertado pode gerar muitas perdas pequenas"
echo "   • Sem stoploss pode funcionar em tendências fortes"
echo ""