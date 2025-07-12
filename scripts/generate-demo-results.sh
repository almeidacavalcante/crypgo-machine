#!/bin/bash

# Quick demo to generate parameter optimization results table
# Uses a smaller parameter set for faster execution

set -e

echo "🎯 Gerando tabela demo de otimização de parâmetros..."
echo "===================================================="

# Store original directory
ORIGINAL_DIR=$(pwd)

# Create results directory
mkdir -p demo_optimization_results
cd demo_optimization_results

# Clear previous results
rm -f *.json
rm -f demo_summary.txt

# Demo parameters (smaller set for speed)
SYMBOLS=("SOLBRL")
FAST_WINDOWS=(3 5)
SLOW_WINDOWS=(10 15)
MIN_PROFITS=(1.0 2.0)
MIN_SPREADS=(0.3 0.5)
AMOUNTS=(1000 2000)

# Fixed parameters
START_DATE="2024-06-01"
END_DATE="2024-06-07"
CAPITAL=5000
INTERVAL="1h"
FEES=0.02
CURRENCY="BRL"

# Initialize results table
echo "# DEMO OTIMIZAÇÃO DE PARÂMETROS - RESULTADOS" > demo_summary.txt
echo "## Período: $START_DATE a $END_DATE (dados reais Binance)" >> demo_summary.txt
echo "## Capital inicial: $CAPITAL BRL" >> demo_summary.txt
echo "" >> demo_summary.txt
printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s\n" \
    "Symbol" "Fast" "Slow" "MinProfit" "MinSpread" "Amount" "ROI%" "Trades" "WinRate%" "MaxDD%" >> demo_summary.txt
echo "===================================================================================" >> demo_summary.txt

# Counters
TOTAL_TESTS=0
COMPLETED_TESTS=0
BEST_ROI=-999
BEST_CONFIG=""

# Calculate total number of tests
for symbol in "${SYMBOLS[@]}"; do
    for fast in "${FAST_WINDOWS[@]}"; do
        for slow in "${SLOW_WINDOWS[@]}"; do
            for min_profit in "${MIN_PROFITS[@]}"; do
                for min_spread in "${MIN_SPREADS[@]}"; do
                    for amount in "${AMOUNTS[@]}"; do
                        if [ $fast -lt $slow ]; then
                            TOTAL_TESTS=$((TOTAL_TESTS + 1))
                        fi
                    done
                done
            done
        done
    done
done

echo "📊 Total de testes: $TOTAL_TESTS"
echo ""

# Function to run a single backtest
run_test() {
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
    local filename="demo_${symbol}_f${fast}_s${slow}_p${min_profit}_sp${min_spread}_a${amount}.json"
    
    echo "🔄 [$((COMPLETED_TESTS + 1))/$TOTAL_TESTS] $symbol Fast=$fast Slow=$slow Profit=$min_profit% Spread=$min_spread% Amount=$amount"
    
    # Build and run command
    local cmd="cd $ORIGINAL_DIR && go run cmd/backtest/main.go"
    cmd="$cmd -start=$START_DATE -end=$END_DATE"
    cmd="$cmd -symbol=$symbol -fast=$fast -slow=$slow"
    cmd="$cmd -capital=$CAPITAL -amount=$amount"
    cmd="$cmd -min-profit=$min_profit -min-spread=$min_spread"
    cmd="$cmd -interval=$INTERVAL -fees=$FEES"
    cmd="$cmd -quantity=$quantity -currency=$CURRENCY"
    cmd="$cmd -output=demo_optimization_results/$filename -quiet"
    
    # Run backtest
    if eval "$cmd" > /dev/null 2>&1; then
        # Extract results from JSON
        if [ -f "$filename" ]; then
            local roi=$(cat "$filename" | jq -r '.roi // 0')
            local trades=$(cat "$filename" | jq -r '.total_trades // 0')
            local win_rate=$(cat "$filename" | jq -r '.win_rate // 0')
            local max_dd=$(cat "$filename" | jq -r '.max_drawdown // 0')
            
            # Add to results table
            printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8.2f %-8s %-8.1f %-8.2f\n" \
                "$symbol" "$fast" "$slow" "$min_profit" "$min_spread" "$amount" \
                "$roi" "$trades" "$win_rate" "$max_dd" >> demo_summary.txt
            
            # Check if this is the best ROI so far
            if (( $(echo "$roi > $BEST_ROI" | bc -l) )); then
                BEST_ROI=$roi
                BEST_CONFIG="$symbol Fast=$fast Slow=$slow Profit=$min_profit% Spread=$min_spread% Amount=$amount"
            fi
            
            echo "   ✅ ROI: $roi% | Trades: $trades | Win Rate: $win_rate%"
        else
            echo "   ❌ Arquivo não gerado"
            printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s\n" \
                "$symbol" "$fast" "$slow" "$min_profit" "$min_spread" "$amount" \
                "ERROR" "ERROR" "ERROR" "ERROR" >> demo_summary.txt
        fi
    else
        echo "   ❌ Falha na execução"
        printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s\n" \
            "$symbol" "$fast" "$slow" "$min_profit" "$min_spread" "$amount" \
            "FAILED" "FAILED" "FAILED" "FAILED" >> demo_summary.txt
    fi
    
    COMPLETED_TESTS=$((COMPLETED_TESTS + 1))
    echo ""
}

# Main loop
for symbol in "${SYMBOLS[@]}"; do
    for fast in "${FAST_WINDOWS[@]}"; do
        for slow in "${SLOW_WINDOWS[@]}"; do
            if [ $fast -lt $slow ]; then
                for min_profit in "${MIN_PROFITS[@]}"; do
                    for min_spread in "${MIN_SPREADS[@]}"; do
                        for amount in "${AMOUNTS[@]}"; do
                            run_test "$symbol" "$fast" "$slow" "$min_profit" "$min_spread" "$amount"
                        done
                    done
                done
            fi
        done
    done
done

# Generate summary
echo "" >> demo_summary.txt
echo "===================================================================================" >> demo_summary.txt
echo "## RESUMO FINAL" >> demo_summary.txt
echo "" >> demo_summary.txt
echo "🏆 MELHOR CONFIGURAÇÃO:" >> demo_summary.txt
echo "   ROI: $BEST_ROI%" >> demo_summary.txt
echo "   Parâmetros: $BEST_CONFIG" >> demo_summary.txt
echo "" >> demo_summary.txt

# Sort results by ROI (exclude ERROR/FAILED entries)
echo "## TOP CONFIGURAÇÕES (ordenadas por ROI):" >> demo_summary.txt
echo "" >> demo_summary.txt
printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s\n" \
    "Symbol" "Fast" "Slow" "MinProfit" "MinSpread" "Amount" "ROI%" "Trades" "WinRate%" "MaxDD%" >> demo_summary.txt
echo "===================================================================================" >> demo_summary.txt

# Extract and sort numeric results
grep -v "ERROR\|FAILED" demo_summary.txt | grep -E "^[A-Z]" | sort -k7 -nr >> demo_summary.txt

echo "✅ Demo concluído!"
echo ""
echo "📊 RESULTADOS:"
echo "   🏆 Melhor ROI: $BEST_ROI% ($BEST_CONFIG)"
echo ""
echo "🔍 Ver tabela completa:"
echo "   cat demo_optimization_results/demo_summary.txt"
echo ""
echo "📈 TABELA DE RESULTADOS:"
echo "========================================"
cat demo_summary.txt
echo "========================================"