#!/bin/bash

# Parameter Optimization Script for Trading Bot Backtest
# This script runs multiple backtests with different parameter combinations
# and generates a comparison table to find the best performing configuration

set -e

echo "🚀 Iniciando otimização de parâmetros do trading bot..."
echo "========================================================="

# Check dependencies
echo "🔍 Verificando dependências..."
if ! command -v jq &> /dev/null; then
    echo "❌ Erro: jq não está instalado. Instale com: brew install jq"
    exit 1
fi

if ! command -v bc &> /dev/null; then
    echo "❌ Erro: bc não está instalado. Instale com: brew install bc"
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
mkdir -p optimization_results
cd optimization_results

# Clear previous results
rm -f *.json
rm -f optimization_summary.txt

# Test parameters - feel free to modify these ranges
SYMBOLS=("SOLBRL" "BTCBRL")
FAST_WINDOWS=(3 5 7)
SLOW_WINDOWS=(10 15 20)
MIN_PROFITS=(0.5 1.0 1.5 2.0 3.0)
MIN_SPREADS=(0.2 0.5 1.0)
AMOUNTS=(500 1000 2000)

# Fixed parameters for consistency
START_DATE="2024-05-01"
END_DATE="2024-06-30"
CAPITAL=5000
INTERVAL="1h"
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
echo "# OTIMIZAÇÃO DE PARÂMETROS - RESULTADOS" > optimization_summary.txt
echo "## Período: $START_DATE a $END_DATE" >> optimization_summary.txt
echo "## Capital inicial: $CAPITAL BRL" >> optimization_summary.txt
echo "" >> optimization_summary.txt
printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s %-12s\n" \
    "Symbol" "Fast" "Slow" "MinProfit" "MinSpread" "Amount" "ROI%" "Trades" "WinRate%" "MaxDD%" "Filename" >> optimization_summary.txt
echo "=================================================================================================" >> optimization_summary.txt

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
    local filename="backtest_${symbol}_f${fast}_s${slow}_p${min_profit}_sp${min_spread}_a${amount}.json"
    
    echo "🔄 Teste $((COMPLETED_TESTS + 1))/$TOTAL_TESTS: $symbol Fast=$fast Slow=$slow Profit=$min_profit% Spread=$min_spread% Amount=$amount"
    
    # Build command (run from original directory to find .env file)
    local cmd="cd $ORIGINAL_DIR && go run cmd/backtest/main.go -start=$START_DATE -end=$END_DATE -symbol=$symbol -fast=$fast -slow=$slow -capital=$CAPITAL -amount=$amount -min-profit=$min_profit -min-spread=$min_spread -interval=$INTERVAL -fees=$FEES -quantity=$quantity -currency=$CURRENCY -output=optimization_results/$filename -quiet"
    
    # Run backtest (suppress output for cleaner results)
    if eval "$cmd" > /dev/null 2>&1; then
        
        # Extract results from JSON (check in current directory)
        if [ -f "$filename" ]; then
            local roi=$(cat "$filename" | jq -r '.roi // 0')
            local trades=$(cat "$filename" | jq -r '.total_trades // 0')
            local win_rate=$(cat "$filename" | jq -r '.win_rate // 0')
            local max_dd=$(cat "$filename" | jq -r '.max_drawdown // 0')
            
            # Add to results table
            printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8.2f %-8s %-8.1f %-8.2f %-12s\n" \
                "$symbol" "$fast" "$slow" "$min_profit" "$min_spread" "$amount" \
                "$roi" "$trades" "$win_rate" "$max_dd" "$filename" >> optimization_summary.txt
            
            # Check if this is the best ROI so far
            if (( $(echo "$roi > $BEST_ROI" | bc -l) )); then
                BEST_ROI=$roi
                BEST_CONFIG="$symbol Fast=$fast Slow=$slow Profit=$min_profit% Spread=$min_spread% Amount=$amount"
                BEST_FILE="$filename"
            fi
            
            echo "   ✅ ROI: $roi% | Trades: $trades | Win Rate: $win_rate% | Max DD: $max_dd%"
        else
            echo "   ❌ Falha - arquivo não gerado"
            printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s %-12s\n" \
                "$symbol" "$fast" "$slow" "$min_profit" "$min_spread" "$amount" \
                "ERROR" "ERROR" "ERROR" "ERROR" "NO_FILE" >> optimization_summary.txt
        fi
    else
        echo "   ❌ Falha na execução"
        printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s %-12s\n" \
            "$symbol" "$fast" "$slow" "$min_profit" "$min_spread" "$amount" \
            "FAILED" "FAILED" "FAILED" "FAILED" "EXEC_ERROR" >> optimization_summary.txt
    fi
    
    COMPLETED_TESTS=$((COMPLETED_TESTS + 1))
    echo ""
}

# Main optimization loop
echo "🎯 Executando testes com diferentes combinações de parâmetros..."
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
echo "" >> optimization_summary.txt
echo "=================================================================================================" >> optimization_summary.txt
echo "## RESUMO FINAL" >> optimization_summary.txt
echo "" >> optimization_summary.txt
echo "🏆 MELHOR CONFIGURAÇÃO:" >> optimization_summary.txt
echo "   ROI: $BEST_ROI%" >> optimization_summary.txt
echo "   Parâmetros: $BEST_CONFIG" >> optimization_summary.txt
echo "   Arquivo: $BEST_FILE" >> optimization_summary.txt
echo "" >> optimization_summary.txt

# Sort results by ROI
echo "## TOP 10 MELHORES CONFIGURAÇÕES (por ROI):" >> optimization_summary.txt
echo "" >> optimization_summary.txt
printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s\n" \
    "Symbol" "Fast" "Slow" "MinProfit" "MinSpread" "Amount" "ROI%" "Trades" "WinRate%" "MaxDD%" >> optimization_summary.txt
echo "========================================================================================" >> optimization_summary.txt

# Extract and sort numeric results (exclude ERROR/FAILED entries)
grep -v "ERROR\|FAILED" optimization_summary.txt | grep -E "^[A-Z]" | sort -k7 -nr | head -10 >> optimization_summary.txt

echo "✅ Otimização concluída!"
echo ""
echo "📊 RESULTADOS:"
echo "   📁 Arquivos JSON individuais: optimization_results/*.json"
echo "   📋 Tabela resumo: optimization_results/optimization_summary.txt"
echo "   🏆 Melhor ROI: $BEST_ROI% ($BEST_CONFIG)"
echo ""
echo "🔍 Para ver a tabela completa:"
echo "   cat optimization_results/optimization_summary.txt"
echo ""
echo "💡 Para analisar o melhor resultado:"
echo "   cat optimization_results/$BEST_FILE | jq ."