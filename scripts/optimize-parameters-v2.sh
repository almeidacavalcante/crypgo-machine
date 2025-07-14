#!/bin/bash

# Script de otimização de parâmetros v2 - Com intervalos corrigidos
# Executa backtests com diferentes combinações de parâmetros

echo "🚀 Iniciando Otimização de Parâmetros v2..."
echo "📊 Nova versão com intervalos corrigidos (30m consistentes)"
echo ""

# Configurações base
START_DATE="2024-06-01"
END_DATE="2024-06-30"
CAPITAL=5000.0
TRADE_AMOUNT=2000.0
CURRENCY="BRL"
FEES=0.01
SYMBOL="SOLBRL"

# Arquivo de resultados
RESULTS_FILE="optimization_results/backtest_results_v2_$(date +%Y%m%d_%H%M%S).txt"
mkdir -p optimization_results

echo "📈 OTIMIZAÇÃO DE PARÂMETROS v2 - $(date)" > $RESULTS_FILE
echo "==================================================" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "📋 Configurações Base:" >> $RESULTS_FILE
echo "Período: $START_DATE a $END_DATE" >> $RESULTS_FILE
echo "Capital: $CAPITAL $CURRENCY" >> $RESULTS_FILE
echo "Trade Amount: $TRADE_AMOUNT $CURRENCY" >> $RESULTS_FILE
echo "Fees: $FEES%" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

# Contadores
total_tests=0
successful_tests=0
best_roi=0
best_config=""

# Arrays de parâmetros para testar
fast_windows=(3 5 7 10 12 15)
slow_windows=(10 20 30 40 50 60)
min_profits=(1.0 2.0 3.0 5.0 7.0 10.0)

echo "🔍 Testando ${#fast_windows[@]} fast windows x ${#slow_windows[@]} slow windows x ${#min_profits[@]} min profits = $((${#fast_windows[@]} * ${#slow_windows[@]} * ${#min_profits[@]})) combinações"
echo ""

# Função para executar backtest
run_backtest() {
    local fast=$1
    local slow=$2
    local min_profit=$3
    local symbol=$4
    
    echo "⚡ Testando: Fast=$fast, Slow=$slow, MinProfit=${min_profit}%, Symbol=$symbol"
    
    local output=$(go run cmd/backtest/main.go \
        -start="$START_DATE" \
        -end="$END_DATE" \
        -symbol="$symbol" \
        -fast=$fast \
        -slow=$slow \
        -capital=$CAPITAL \
        -amount=$TRADE_AMOUNT \
        -currency="$CURRENCY" \
        -fees=$FEES \
        -min-profit=$min_profit \
        -quiet 2>&1)
    
    local exit_code=$?
    total_tests=$((total_tests + 1))
    
    if [ $exit_code -eq 0 ]; then
        successful_tests=$((successful_tests + 1))
        
        # Extrair ROI do output
        local roi=$(echo "$output" | grep -o "📊 ROI: [0-9.-]*%" | head -1 | grep -o "[0-9.-]*" | head -1)
        local trades=$(echo "$output" | grep -o "🔄 Total Trades: [0-9]*" | head -1 | grep -o "[0-9]*" | head -1)
        local win_rate=$(echo "$output" | grep -o "🎯 Win Rate: [0-9.-]*%" | head -1 | grep -o "[0-9.-]*" | head -1)
        
        if [ ! -z "$roi" ]; then
            echo "   📊 ROI: ${roi}%, Trades: $trades, Win Rate: ${win_rate}%"
            
            # Salvar resultado
            echo "Fast: $fast, Slow: $slow, MinProfit: ${min_profit}%, Symbol: $symbol | ROI: ${roi}%, Trades: $trades, Win Rate: ${win_rate}%" >> $RESULTS_FILE
            
            # Verificar se é o melhor ROI
            if (( $(echo "$roi > $best_roi" | bc -l) )); then
                best_roi=$roi
                best_config="Fast: $fast, Slow: $slow, MinProfit: ${min_profit}%, Symbol: $symbol"
                echo "   🏆 NOVO CAMPEÃO! ROI: ${roi}%"
            fi
        else
            echo "   ❌ Erro ao extrair ROI"
        fi
    else
        echo "   ❌ Erro na execução: $output"
        echo "ERRO - Fast: $fast, Slow: $slow, MinProfit: ${min_profit}%, Symbol: $symbol | $output" >> $RESULTS_FILE
    fi
    echo ""
}

# Executar testes para SOLBRL
echo "🟡 Testando SOLBRL..."
echo "" >> $RESULTS_FILE
echo "🟡 RESULTADOS SOLBRL:" >> $RESULTS_FILE
echo "=====================" >> $RESULTS_FILE

for fast in "${fast_windows[@]}"; do
    for slow in "${slow_windows[@]}"; do
        # Só testa se slow > fast (lógica das médias móveis)
        if [ $slow -gt $fast ]; then
            for min_profit in "${min_profits[@]}"; do
                run_backtest $fast $slow $min_profit "SOLBRL"
                # Pequena pausa para não sobrecarregar a API
                sleep 1
            done
        fi
    done
done

# Executar testes para BTCBRL
echo "🟠 Testando BTCBRL..."
echo "" >> $RESULTS_FILE
echo "🟠 RESULTADOS BTCBRL:" >> $RESULTS_FILE
echo "=====================" >> $RESULTS_FILE

for fast in "${fast_windows[@]}"; do
    for slow in "${slow_windows[@]}"; do
        if [ $slow -gt $fast ]; then
            for min_profit in "${min_profits[@]}"; do
                run_backtest $fast $slow $min_profit "BTCBRL"
                sleep 1
            done
        fi
    done
done

# Resumo final
echo "" >> $RESULTS_FILE
echo "📊 RESUMO FINAL:" >> $RESULTS_FILE
echo "===============" >> $RESULTS_FILE
echo "Total de testes: $total_tests" >> $RESULTS_FILE
echo "Testes bem-sucedidos: $successful_tests" >> $RESULTS_FILE
echo "Taxa de sucesso: $(echo "scale=2; $successful_tests * 100 / $total_tests" | bc)%" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE
echo "🏆 MELHOR CONFIGURAÇÃO:" >> $RESULTS_FILE
echo "$best_config" >> $RESULTS_FILE
echo "ROI: ${best_roi}%" >> $RESULTS_FILE

echo "✅ Otimização concluída!"
echo "📊 Total de testes: $total_tests"
echo "✅ Testes bem-sucedidos: $successful_tests"
echo "🏆 Melhor ROI: ${best_roi}%"
echo "🏆 Melhor configuração: $best_config"
echo ""
echo "📁 Resultados salvos em: $RESULTS_FILE"

# Mostrar top 10 melhores resultados
echo ""
echo "🏆 TOP 10 MELHORES CONFIGURAÇÕES:"
echo "================================="
grep "ROI:" $RESULTS_FILE | sort -t: -k4 -nr | head -10