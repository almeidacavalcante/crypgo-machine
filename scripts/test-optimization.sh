#!/bin/bash

# Simple test version of optimization script
# Tests just a few parameter combinations to verify functionality

set -e

echo "🧪 Teste simples do sistema de otimização..."
echo "============================================="

# Check dependencies
echo "🔍 Verificando dependências..."
if ! command -v jq &> /dev/null; then
    echo "❌ Erro: jq não está instalado. Instale com: brew install jq"
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ Erro: Go não está instalado"
    exit 1
fi

echo "✅ Dependências verificadas"

# Store original directory
ORIGINAL_DIR=$(pwd)

# Create results directory
mkdir -p test_optimization_results
cd test_optimization_results

# Clear previous results
rm -f *.json
rm -f test_summary.txt

# Test parameters (just a few combinations)
SYMBOLS=("SOLBRL")
FAST_WINDOWS=(3 5)
SLOW_WINDOWS=(20 45)
MIN_PROFITS=(2.0 5.0)
MIN_SPREADS=(0.5)
AMOUNTS=(5000)

# Fixed parameters for consistency
START_DATE="2024-06-01"
END_DATE="2024-07-12"
CAPITAL=5000
INTERVAL="30m"
FEES=0.02
CURRENCY="BRL"

# Counters
TOTAL_TESTS=0
COMPLETED_TESTS=0
BEST_ROI=-999
BEST_CONFIG=""
BEST_FILE=""

# Calculate total number of tests
for symbol in "${SYMBOLS[@]}"; do
    for fast in "${FAST_WINDOWS[@]}"; do
        for slow in "${SLOW_WINDOWS[@]}"; do
            for min_profit in "${MIN_PROFITS[@]}"; do
                for min_spread in "${MIN_SPREADS[@]}"; do
                    for amount in "${AMOUNTS[@]}"; do
                        if [ $fast -lt $slow ]; then  # Only test if fast < slow
                            TOTAL_TESTS=$((TOTAL_TESTS + 1))
                        fi
                    done
                done
            done
        done
    done
done

echo "📊 Total de testes planejados: $TOTAL_TESTS"
echo ""

# Initialize results table
echo "# TESTE DE OTIMIZAÇÃO - RESULTADOS" > test_summary.txt
echo "## Período: $START_DATE a $END_DATE" >> test_summary.txt
echo "## Capital inicial: $CAPITAL BRL" >> test_summary.txt
echo "" >> test_summary.txt
printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s %-12s\n" \
    "Symbol" "Fast" "Slow" "MinProfit" "MinSpread" "Amount" "ROI%" "Trades" "WinRate%" "MaxDD%" "Filename" >> test_summary.txt
echo "=================================================================================================" >> test_summary.txt

# Function to run a single backtest
run_backtest() {
    local symbol=$1
    local fast=$2
    local slow=$3
    local min_profit=$4
    local min_spread=$5
    local amount=$6
    
    # Set quantity based on symbol
    local quantity=0.1
    if [ "$symbol" = "BTCBRL" ]; then
        quantity=0.001
    fi
    
    # Generate filename
    local filename="test_${symbol}_f${fast}_s${slow}_p${min_profit}_sp${min_spread}_a${amount}.json"
    
    echo "🔄 Teste $((COMPLETED_TESTS + 1))/$TOTAL_TESTS: $symbol Fast=$fast Slow=$slow Profit=$min_profit% Spread=$min_spread% Amount=$amount"
    
    # Build command (run from original directory to find .env file)
    local cmd="cd $ORIGINAL_DIR && go run cmd/backtest/main.go"
    cmd="$cmd -start=$START_DATE -end=$END_DATE"
    cmd="$cmd -symbol=$symbol -fast=$fast -slow=$slow"
    cmd="$cmd -capital=$CAPITAL -amount=$amount"
    cmd="$cmd -min-profit=$min_profit -min-spread=$min_spread"
    cmd="$cmd -interval=$INTERVAL -fees=$FEES"
    cmd="$cmd -quantity=$quantity -currency=$CURRENCY"
    cmd="$cmd -output=test_optimization_results/$filename -quiet"
    
    echo "   🐛 Comando: $cmd"
    echo ""
    
    # Run backtest
    if eval "$cmd"; then
        echo ""
        echo "   ✅ Comando executado com sucesso"
        
        # Extract results from JSON
        if [ -f "$filename" ]; then
            local roi=$(cat "$filename" | jq -r '.roi // 0')
            local trades=$(cat "$filename" | jq -r '.total_trades // 0')
            local win_rate=$(cat "$filename" | jq -r '.win_rate // 0')
            local max_dd=$(cat "$filename" | jq -r '.max_drawdown // 0')
            
            # Add to results table
            printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8.2f %-8s %-8.1f %-8.2f %-12s\n" \
                "$symbol" "$fast" "$slow" "$min_profit" "$min_spread" "$amount" \
                "$roi" "$trades" "$win_rate" "$max_dd" "$filename" >> test_summary.txt
            
            # Check if this is the best ROI so far
            if (( $(echo "$roi > $BEST_ROI" | bc -l) )); then
                BEST_ROI=$roi
                BEST_CONFIG="$symbol Fast=$fast Slow=$slow Profit=$min_profit% Spread=$min_spread% Amount=$amount"
                BEST_FILE="$filename"
            fi
            
            echo "   📊 ROI: $roi% | Trades: $trades | Win Rate: $win_rate% | Max DD: $max_dd%"
        else
            echo "   ❌ Falha - arquivo JSON não foi gerado"
            printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s %-12s\n" \
                "$symbol" "$fast" "$slow" "$min_profit" "$min_spread" "$amount" \
                "NO_FILE" "NO_FILE" "NO_FILE" "NO_FILE" "NO_FILE" >> test_summary.txt
        fi
    else
        echo ""
        echo "   ❌ Falha na execução do comando"
        printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s %-12s\n" \
            "$symbol" "$fast" "$slow" "$min_profit" "$min_spread" "$amount" \
            "FAILED" "FAILED" "FAILED" "FAILED" "EXEC_ERROR" >> test_summary.txt
    fi
    
    COMPLETED_TESTS=$((COMPLETED_TESTS + 1))
    echo ""
    echo "=================================================="
    echo ""
}

# Main test loop
echo "🎯 Executando testes com combinações limitadas..."
echo ""

for symbol in "${SYMBOLS[@]}"; do
    for fast in "${FAST_WINDOWS[@]}"; do
        for slow in "${SLOW_WINDOWS[@]}"; do
            if [ $fast -lt $slow ]; then  # Only test if fast < slow
                for min_profit in "${MIN_PROFITS[@]}"; do
                    for min_spread in "${MIN_SPREADS[@]}"; do
                        for amount in "${AMOUNTS[@]}"; do
                            run_backtest "$symbol" "$fast" "$slow" "$min_profit" "$min_spread" "$amount"
                        done
                    done
                done
            fi
        done
    done
done

# Generate final summary
echo "" >> test_summary.txt
echo "=================================================================================================" >> test_summary.txt
echo "## RESUMO FINAL" >> test_summary.txt
echo "" >> test_summary.txt
echo "🏆 MELHOR CONFIGURAÇÃO:" >> test_summary.txt
echo "   ROI: $BEST_ROI%" >> test_summary.txt
echo "   Parâmetros: $BEST_CONFIG" >> test_summary.txt
echo "   Arquivo: $BEST_FILE" >> test_summary.txt
echo "" >> test_summary.txt

echo "✅ Teste concluído!"
echo ""
echo "📊 RESULTADOS:"
echo "   📁 Arquivos JSON: test_optimization_results/*.json"
echo "   📋 Tabela resumo: test_optimization_results/test_summary.txt"
echo "   🏆 Melhor ROI: $BEST_ROI% ($BEST_CONFIG)"
echo ""
echo "🔍 Para ver a tabela completa:"
echo "   cat test_optimization_results/test_summary.txt"
echo ""
if [ "$BEST_FILE" != "" ]; then
    echo "💡 Para analisar o melhor resultado:"
    echo "   cat test_optimization_results/$BEST_FILE | jq ."
fi