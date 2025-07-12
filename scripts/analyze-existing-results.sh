#!/bin/bash

# Script to analyze existing backtest results in optimization_results/
# This will read the JSON files that were already generated and create a summary table

set -e

echo "📊 Analisando resultados existentes de otimização..."
echo "=================================================="

# Store original directory
ORIGINAL_DIR=$(pwd)

# Check if results directory exists
if [ ! -d "optimization_results" ]; then
    echo "❌ Diretório optimization_results não encontrado"
    exit 1
fi

cd optimization_results

# Count JSON files
JSON_COUNT=$(ls -1 *.json 2>/dev/null | wc -l)
echo "📁 Encontrados $JSON_COUNT arquivos JSON de resultados"

if [ $JSON_COUNT -eq 0 ]; then
    echo "❌ Nenhum arquivo de resultado encontrado"
    exit 1
fi

# Initialize results table
echo "# ANÁLISE DE RESULTADOS EXISTENTES" > analysis_summary.txt
echo "## Arquivos analisados: $JSON_COUNT" >> analysis_summary.txt
echo "" >> analysis_summary.txt
printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s %-20s\n" \
    "Symbol" "Fast" "Slow" "MinProfit" "MinSpread" "Amount" "ROI%" "Trades" "WinRate%" "MaxDD%" "Filename" >> analysis_summary.txt
echo "=================================================================================================================" >> analysis_summary.txt

BEST_ROI=-999
BEST_CONFIG=""
BEST_FILE=""
PROCESSED=0

echo ""
echo "🔍 Analisando arquivos JSON..."

# Process each JSON file
for file in *.json; do
    if [ "$file" = "*.json" ]; then
        continue  # No files found
    fi
    
    # Skip if not a backtest result file
    if [[ ! "$file" =~ ^backtest_.*\.json$ ]]; then
        continue
    fi
    
    PROCESSED=$((PROCESSED + 1))
    echo "   [$PROCESSED/$JSON_COUNT] Processando: $file"
    
    # Extract parameters from filename
    # Format: backtest_SYMBOL_fFAST_sSLOW_pPROFIT_spSPREAD_aAMOUNT.json
    if [[ "$file" =~ backtest_([A-Z]+)_f([0-9]+)_s([0-9]+)_p([0-9.]+)_sp([0-9.]+)_a([0-9]+)\.json ]]; then
        symbol="${BASH_REMATCH[1]}"
        fast="${BASH_REMATCH[2]}"
        slow="${BASH_REMATCH[3]}"
        min_profit="${BASH_REMATCH[4]}"
        min_spread="${BASH_REMATCH[5]}"
        amount="${BASH_REMATCH[6]}"
        
        # Extract results from JSON
        if [ -f "$file" ]; then
            roi=$(cat "$file" | jq -r '.roi // 0')
            trades=$(cat "$file" | jq -r '.total_trades // 0')
            win_rate=$(cat "$file" | jq -r '.win_rate // 0')
            max_dd=$(cat "$file" | jq -r '.max_drawdown // 0')
            
            # Add to results table
            printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8.2f %-8s %-8.1f %-8.2f %-20s\n" \
                "$symbol" "$fast" "$slow" "$min_profit" "$min_spread" "$amount" \
                "$roi" "$trades" "$win_rate" "$max_dd" "$file" >> analysis_summary.txt
            
            # Check if this is the best ROI so far
            if (( $(echo "$roi > $BEST_ROI" | bc -l) )); then
                BEST_ROI=$roi
                BEST_CONFIG="$symbol Fast=$fast Slow=$slow Profit=${min_profit}% Spread=${min_spread}% Amount=$amount"
                BEST_FILE="$file"
            fi
            
            echo "      ✅ ROI: $roi% | Trades: $trades | Win Rate: $win_rate%"
        else
            echo "      ❌ Arquivo não encontrado"
        fi
    else
        echo "      ⚠️ Nome de arquivo não reconhecido: $file"
    fi
done

# Generate summary
echo "" >> analysis_summary.txt
echo "=================================================================================================================" >> analysis_summary.txt
echo "## RESUMO FINAL" >> analysis_summary.txt
echo "" >> analysis_summary.txt
echo "🏆 MELHOR CONFIGURAÇÃO:" >> analysis_summary.txt
echo "   ROI: $BEST_ROI%" >> analysis_summary.txt
echo "   Parâmetros: $BEST_CONFIG" >> analysis_summary.txt
echo "   Arquivo: $BEST_FILE" >> analysis_summary.txt
echo "" >> analysis_summary.txt

# Sort results by ROI
echo "## TOP 10 CONFIGURAÇÕES (ordenadas por ROI):" >> analysis_summary.txt
echo "" >> analysis_summary.txt
printf "%-8s %-4s %-4s %-8s %-8s %-8s %-8s %-8s %-8s %-8s\n" \
    "Symbol" "Fast" "Slow" "MinProfit" "MinSpread" "Amount" "ROI%" "Trades" "WinRate%" "MaxDD%" >> analysis_summary.txt
echo "========================================================================================" >> analysis_summary.txt

# Extract and sort numeric results
grep -v "^#\|^$\|^=" analysis_summary.txt | grep -E "^[A-Z]" | sort -k7 -nr | head -10 >> analysis_summary.txt

echo ""
echo "✅ Análise concluída!"
echo ""
echo "📊 RESULTADOS:"
echo "   📁 Arquivos processados: $PROCESSED"
echo "   🏆 Melhor ROI: $BEST_ROI% ($BEST_CONFIG)"
echo "   📋 Relatório: optimization_results/analysis_summary.txt"
echo ""
echo "🔍 TOP 5 RESULTADOS:"
echo "========================================================================================"
grep -v "^#\|^$\|^=" analysis_summary.txt | grep -E "^[A-Z]" | sort -k7 -nr | head -5
echo "========================================================================================"
echo ""
echo "💡 Para ver detalhes do melhor resultado:"
echo "   cat optimization_results/$BEST_FILE | jq ."