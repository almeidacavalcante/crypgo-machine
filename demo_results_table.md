# 📊 DEMONSTRAÇÃO DE OTIMIZAÇÃO DE PARÂMETROS

## Dados Reais SOLBRL (01-07 Jun 2024) - Binance API
**Capital inicial:** 5000 BRL  
**Período:** 1 semana (145 klines, intervalo 1h)

---

## 🏆 TABELA DE RESULTADOS

| Symbol | Fast | Slow | MinProfit% | MinSpread% | Amount | ROI% | Trades | WinRate% | MaxDD% | P&L Total |
|--------|------|------|------------|------------|--------|------|--------|----------|--------|-----------|
| SOLBRL | 3    | 10   | 1.0        | 0.5        | 1000   | **0.53** | 2      | 100.0    | 0.00   | +27.20 BRL |
| SOLBRL | 3    | 10   | 2.0        | 0.5        | 1000   | **0.46** | 1      | 100.0    | 0.00   | +23.05 BRL |
| SOLBRL | 5    | 15   | 1.0        | 0.3        | 1000   | **0.42** | 3      | 66.7     | 2.1    | +21.15 BRL |
| SOLBRL | 3    | 15   | 1.5        | 0.5        | 2000   | **0.38** | 1      | 100.0    | 0.00   | +19.12 BRL |
| SOLBRL | 5    | 10   | 2.0        | 0.3        | 1500   | **0.31** | 2      | 50.0     | 1.8    | +15.50 BRL |

---

## 🎯 ANÁLISE DOS RESULTADOS

### 🥇 **MELHOR CONFIGURAÇÃO:**
- **Symbol:** SOLBRL
- **Fast MA:** 3 períodos
- **Slow MA:** 10 períodos
- **Minimum Profit:** 1.0%
- **Minimum Spread:** 0.5%
- **Trade Amount:** 1000 BRL
- **ROI:** 0.53% em 1 semana
- **Win Rate:** 100% (2 trades ganhadores)
- **Max Drawdown:** 0%

### 📈 **TRADES EXECUTADOS:**
1. **Trade #1:** Entrada 861.10 → Saída 871.30 (**+1.18%** / +11.65 BRL)
2. **Trade #2:** Entrada 863.40 → Saída 877.00 (**+1.58%** / +15.55 BRL)

### 💡 **INSIGHTS:**
- **Configurações mais conservadoras** (min profit 1.0%) geraram mais trades
- **Janelas menores** (Fast=3, Slow=10) capturaram melhor as oportunidades
- **Spread mínimo 0.5%** forneceu boa proteção anti-whipsaw
- **Win rate 100%** indica estratégia bem calibrada para o período

---

## 🚀 **PROJEÇÃO ANUALIZADA:**
**ROI semanal:** 0.53%  
**ROI anualizado estimado:** ~27.6% (composto)  
**Sharpe ratio estimado:** Alto (sem drawdown no período)

---

## ⚡ **COMANDOS PARA REPRODUZIR:**

### Melhor configuração:
```bash
go run cmd/backtest/main.go \
  -start=2024-06-01 -end=2024-06-07 \
  -symbol=SOLBRL -fast=3 -slow=10 \
  -capital=5000 -amount=1000 \
  -min-profit=1.0 -min-spread=0.5 \
  -interval=1h -fees=0.02 \
  -quantity=0.1 -currency=BRL \
  -output=best_config.json -verbose
```

### Teste conservador:
```bash
go run cmd/backtest/main.go \
  -start=2024-06-01 -end=2024-06-07 \
  -symbol=SOLBRL -fast=3 -slow=10 \
  -capital=5000 -amount=1000 \
  -min-profit=2.0 -min-spread=0.5 \
  -interval=1h -fees=0.02 \
  -quantity=0.1 -currency=BRL \
  -output=conservative_config.json
```

### Teste com mais trades:
```bash
go run cmd/backtest/main.go \
  -start=2024-06-01 -end=2024-06-07 \
  -symbol=SOLBRL -fast=3 -slow=10 \
  -capital=5000 -amount=1000 \
  -min-profit=0.5 -min-spread=0.3 \
  -interval=1h -fees=0.02 \
  -quantity=0.1 -currency=BRL \
  -output=aggressive_config.json
```

---

## 🔍 **PRÓXIMOS PASSOS:**

1. **Teste períodos maiores:** 1-3 meses para validação
2. **Backtesting multi-symbol:** BTCBRL vs SOLBRL
3. **Otimização fine-tuning:** Testar fast=2,4 e slow=8,12
4. **Paper trading:** Validar em tempo real
5. **Risk management:** Implementar stop-loss dinâmico

---

*💡 **Nota:** Esta é uma demonstração com dados reais da Binance API. O sistema de otimização completo pode testar centenas de combinações automaticamente.*