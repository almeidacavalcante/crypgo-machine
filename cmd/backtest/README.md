# Backtest Command Line Tool

Este comando permite executar backtests usando dados reais do Binance com a mesma lógica de trading usado no sistema live.

## Características

- ✅ **Reutiliza a lógica de trading real**: Usa exatamente o mesmo código do `StartTradingBotUseCase`
- ✅ **Dados reais do Binance**: Busca dados históricos direto da API do Binance
- ✅ **Suporte ao minimum profit threshold**: Testa a funcionalidade de lucro mínimo implementada
- ✅ **Métricas completas**: ROI, win rate, drawdown, P&L detalhado
- ✅ **Interface simples**: Comando de linha com flags intuitivas

## Uso Básico

```bash
# Exemplo simples - backtest de 30 dias
go run cmd/backtest/main.go \
  -start=2024-01-01 \
  -end=2024-01-31 \
  -symbol=BTCBRL \
  -fast=5 \
  -slow=10 \
  -min-profit=2.0

# Backtest com configurações customizadas
go run cmd/backtest/main.go \
  -start=2024-06-01 \
  -end=2024-06-30 \
  -symbol=SOLBRL \
  -fast=3 \
  -slow=7 \
  -capital=5000 \
  -amount=500 \
  -fees=0.15 \
  -min-profit=1.5 \
  -interval=4h \
  -output=results.json
```

## Parâmetros

| Flag | Descrição | Padrão | Obrigatório |
|------|-----------|--------|-------------|
| `-start` | Data de início (YYYY-MM-DD) | - | ✅ |
| `-end` | Data de fim (YYYY-MM-DD) | - | ✅ |
| `-symbol` | Par de trading | BTCBRL | ❌ |
| `-strategy` | Estratégia de trading | MovingAverage | ❌ |
| `-fast` | Janela rápida MA | 5 | ❌ |
| `-slow` | Janela lenta MA | 10 | ❌ |
| `-capital` | Capital inicial | 1000.0 | ❌ |
| `-amount` | Valor por trade | 100.0 | ❌ |
| `-fees` | Taxa de trading (%) | 0.1 | ❌ |
| `-min-profit` | Lucro mínimo (%) | 2.0 | ❌ |
| `-interval` | Intervalo das velas | 1h | ❌ |
| `-output` | Arquivo de saída JSON | - | ❌ |

## Configuração das Credenciais

### Opção 1: Variáveis de Ambiente (Recomendado)
```bash
export BINANCE_API_KEY="sua_api_key"
export BINANCE_SECRET_KEY="sua_secret_key"
```

### Opção 2: Flags da Linha de Comando
```bash
go run cmd/backtest/main.go \
  -api-key="sua_api_key" \
  -secret-key="sua_secret_key" \
  -start=2024-01-01 \
  -end=2024-01-31
```

## Exemplo de Saída

```
🚀 Starting backtest with configuration:
   Symbol: BTCBRL
   Strategy: MovingAverage (Fast: 5, Slow: 10)
   Period: 2024-01-01 to 2024-01-31
   Initial Capital: 1000.00 BRL
   Trade Amount: 100.00 BRL
   Minimum Profit Threshold: 2.00%
   
📊 Loaded 744 klines for backtesting BTCBRL from 2024-01-01 to 2024-01-31
🚀 Starting backtest simulation...
📈 Progress: 10.0% (74/744 candles)
📈 Progress: 20.0% (149/744 candles)
...

📈 BACKTEST SUMMARY:
   💰 Total P&L: 87.45 BRL
   📊 ROI: 8.75%
   🎯 Win Rate: 66.67%
   🔄 Total Trades: 12
   ✅ Winning: 8 | ❌ Losing: 4
   📉 Max Drawdown: 3.21%
   💸 Trading Fees: 2.40 BRL

📊 DETAILED TRADE ANALYSIS:
   Trade #  | Entry Price | Exit Price |    P&L    |   P&L%   | Duration
   ---------|-------------|------------|-----------|---------|----------
          1 |   240000.00 |  245200.00 |     20.67 |    2.17% |     72h0m0s
          2 |   245800.00 |  251140.00 |     21.18 |    2.17% |     48h0m0s
          3 |   250900.00 |  248375.00 |    -10.06 |   -1.01% |     24h0m0s
```

## Diferenças do Backtest Anterior

O sistema anterior (`BacktestStrategyUseCase`) tinha várias limitações:

1. **Código duplicado**: Reimplementava a lógica de estratégia
2. **Sincronização manual**: Estado do bot tinha que ser sincronizado manualmente
3. **Risco de divergência**: Lógica diferente do trading real
4. **Manutenção dupla**: Mudanças precisavam ser aplicadas em dois lugares

Este novo sistema resolve todos esses problemas reutilizando exatamente a mesma lógica do trading real.

## Como Funciona

1. **Busca dados históricos** do Binance para o período especificado
2. **Cria um bot virtual** com a estratégia configurada
3. **Usa as mesmas abstrações** (`MarketDataSource` e `TradingExecutionContext`)
4. **Executa a lógica de trading** candle por candle
5. **Simula as operações** e calcula métricas

O resultado é um backtest que reflete exatamente como o bot se comportaria em produção.