# 🤖 N8N Automation Ideas para CrypGo Trading Bot

## 📋 Overview

Este documento apresenta ideias e casos de uso para automações usando N8N no contexto do sistema de trading de criptomoedas CrypGo Machine. O N8N permite criar workflows visuais que conectam diferentes serviços e APIs para automatizar processos complexos.

## 🔗 Informações de Acesso

- **URL**: http://31.97.249.4:8081/
- **Credenciais**: admin / CrypGoN8N2024!
- **Documentação**: [N8N_SETUP.md](./N8N_SETUP.md)

## 🎯 Categorias de Automação

### 1. 📢 Notificações e Alertas

#### A. Notificações de Trading em Tempo Real
- **Trigger**: Webhook do CrypGo quando há BUY/SELL
- **Ações**:
  - Enviar mensagem no Telegram com detalhes da operação
  - Post no Discord com embed rico (preço, símbolo, lucro/prejuízo)
  - Email formatado para operações importantes
  - SMS para alertas críticos (perdas > 5%)

#### B. Relatórios Diários/Semanais
- **Trigger**: Cron schedule (diário 08:00, semanal domingo)
- **Dados coletados**:
  - Performance geral dos bots
  - P&L do período
  - Trades executados
  - Símbolos mais rentáveis
- **Envio**: Email formatado, Google Sheets, Slack

#### C. Alertas de Performance
- **Triggers**:
  - Bot com prejuízo > X% em Y horas
  - Bot parado inesperadamente
  - Falha de conexão com Binance
  - Volume de trades muito baixo
- **Ações**:
  - Notificação urgente no WhatsApp
  - Log em sistema de monitoramento
  - Parar bots automaticamente se necessário

### 2. 📊 Análise e Reporting

#### A. Dashboard Automático no Google Sheets
- **Workflow**:
  1. Coletar dados via API do CrypGo a cada hora
  2. Processar métricas (ROI, Sharpe Ratio, Max Drawdown)
  3. Atualizar Google Sheets com dados formatados
  4. Gerar gráficos automáticos
  5. Compartilhar relatório via email

#### B. Análise de Market Sentiment
- **Dados coletados**:
  - Fear & Greed Index
  - Twitter sentiment para Bitcoin
  - News sentiment (CoinDesk, CoinTelegraph)
  - Reddit r/cryptocurrency sentiment
- **Ação**: Ajustar agressividade dos bots baseado no sentiment

#### C. Comparação com Benchmarks
- **Workflow**:
  1. Coletar performance dos bots
  2. Comparar com índices (BTC, ETH, DeFi)
  3. Calcular alpha e beta
  4. Gerar relatório de performance relativa

### 3. 🔄 Automação de Trading

#### A. Risk Management Automático
- **Monitoramento contínuo**:
  - Stop loss por bot individual
  - Stop loss por portfólio total
  - Correlação entre posições
  - Exposição máxima por ativo
- **Ações automáticas**:
  - Parar bots em condições adversas
  - Reduzir tamanho de posições
  - Diversificar automaticamente

#### B. Rebalanceamento de Portfólio
- **Trigger**: Weekly/Monthly
- **Workflow**:
  1. Analisar performance por símbolo
  2. Calcular alocação ótima
  3. Ajustar `trade_amount` dos bots
  4. Realocar capital entre estratégias

#### C. Otimização de Parâmetros
- **Backtesting automático**:
  1. Rodar backtests com diferentes parâmetros
  2. Comparar resultados (Sharpe, ROI, Max DD)
  3. Sugerir novos parâmetros via email
  4. Aplicar automaticamente se aprovado

### 4. 🔗 Integrações Externas

#### A. Social Trading
- **Copiar sinais de traders profissionais**:
  1. Monitorar Twitter de traders específicos
  2. Parsear sinais de BUY/SELL
  3. Executar trades similares nos bots
  4. Avaliar performance dos sinais

#### B. Economic Calendar Integration
- **Workflow**:
  1. Monitor de eventos econômicos importantes
  2. Pausar trading antes de anúncios do Fed
  3. Ajustar risk management em dias voláteis
  4. Resume trading após eventos

#### C. DeFi Integration
- **Yield farming automático**:
  1. Monitorar APY de diferentes protocolos
  2. Mover fundos idle para yield farming
  3. Compound rewards automaticamente
  4. Return to trading quando necessário

### 5. 🛡️ Monitoramento e Manutenção

#### A. Health Check Avançado
- **Monitoramento**:
  - API response times
  - Database performance
  - Memory/CPU usage
  - Network connectivity
- **Alertas**: Telegram, email, PagerDuty

#### B. Backup Automático
- **Workflow diário**:
  1. Backup database
  2. Export de configurações
  3. Upload para Google Drive/AWS S3
  4. Verificar integridade do backup
  5. Notificar status

#### C. Log Analysis
- **Workflow**:
  1. Analisar logs em busca de padrões
  2. Detectar anomalias
  3. Identificar oportunidades de otimização
  4. Relatório semanal de insights

### 6. 📈 Market Analysis Automations

#### A. Technical Analysis Alerts
- **Indicadores monitorados**:
  - RSI divergence
  - Support/Resistance breaks
  - Volume anomalies
  - Moving average crossovers
- **Ação**: Adjust bot aggressiveness

#### B. Correlation Analysis
- **Workflow**:
  1. Calculate correlation matrix
  2. Identify highly correlated pairs
  3. Adjust position sizing
  4. Suggest diversification

#### C. Volatility Regime Detection
- **Workflow**:
  1. Calculate rolling volatility
  2. Detect regime changes
  3. Adjust strategy parameters
  4. Switch strategies if needed

## 🚀 Workflows Prioritários para Implementação

### Fase 1 - Básico (Semana 1-2)
1. **Notificações Telegram** para trades BUY/SELL
2. **Relatório diário** via email
3. **Health check** básico dos containers

### Fase 2 - Intermediário (Semana 3-4)
1. **Google Sheets integration** para tracking
2. **Risk management** alerts
3. **Performance benchmarking**

### Fase 3 - Avançado (Semana 5-8)
1. **Market sentiment** integration
2. **Automated backtesting**
3. **Portfolio rebalancing**

## 🔧 APIs e Webhooks Necessários

### CrypGo Machine APIs
```http
GET /api/v1/trading/list          # Lista bots
GET /api/v1/trading/logs          # Logs de decisões
GET /api/v1/health               # Health check
POST /webhook/trading-events      # Webhook para eventos
```

### External APIs Sugeridas
- **Telegram Bot API**: Notificações
- **Google Sheets API**: Relatórios
- **CoinGecko API**: Market data
- **Fear & Greed Index API**: Sentiment
- **Twitter API**: Social sentiment
- **SendGrid**: Email notifications

## 📝 Estrutura de Dados para Webhooks

### Trading Event Webhook
```json
{
  "event_type": "trade_executed",
  "timestamp": "2025-07-19T10:30:00Z",
  "bot_id": "uuid-here",
  "symbol": "BTCBRL",
  "decision": "BUY",
  "price": 350000.00,
  "quantity": 0.001,
  "entry_price": 345000.00,
  "profit_loss": 5.00,
  "strategy": "MovingAverage",
  "reason": "FastMA crossed above SlowMA"
}
```

### Bot Status Webhook
```json
{
  "event_type": "bot_status_change",
  "timestamp": "2025-07-19T10:30:00Z",
  "bot_id": "uuid-here",
  "old_status": "running",
  "new_status": "stopped",
  "reason": "Manual stop",
  "is_positioned": true
}
```

## 🎮 Casos de Uso Específicos

### Caso 1: "Smart Notification System"
**Objetivo**: Notificar apenas trades importantes
**Lógica**: 
- BUY/SELL sempre notifica
- HOLD só notifica se há mudança significativa no preço
- Agrupar múltiplas notificações em 5 minutos

### Caso 2: "Performance Leaderboard"
**Objetivo**: Ranking dos bots por performance
**Workflow**:
1. Calcular ROI de cada bot
2. Rankear por performance
3. Postar ranking diário no Discord
4. Highlightar top 3 e bottom 3

### Caso 3: "Emergency Stop System"
**Objetivo**: Parar todos os bots em crash do mercado
**Trigger**: BTC drop > 5% in 1 hour
**Ação**: Stop all bots + notification

### Caso 4: "Profit Taking Automation"
**Objetivo**: Take profit automático em altas
**Lógica**: Se profit > 10%, reduzir position size em 50%

## 🔍 Métricas para Monitoramento

### Performance Metrics
- ROI por bot/período
- Sharpe Ratio
- Maximum Drawdown
- Win Rate
- Average Profit/Loss per Trade

### Operational Metrics
- API Response Time
- Error Rate
- Uptime
- Memory/CPU Usage
- Trade Execution Latency

### Risk Metrics
- Value at Risk (VaR)
- Portfolio Correlation
- Maximum Single Position
- Leverage Ratio
- Liquidity Risk

## 📚 Recursos de Aprendizado

### N8N Workflows para Trading
- [Official N8N Trading Docs](https://docs.n8n.io/)
- [Community Trading Workflows](https://n8n.io/workflows/)
- [Crypto Trading Automation Examples](https://github.com/n8n-io/n8n)

### APIs Úteis
- [Binance API Docs](https://binance-docs.github.io/apidocs/)
- [CoinGecko API](https://www.coingecko.com/en/api)
- [Telegram Bot API](https://core.telegram.org/bots/api)

## 🎯 Próximos Passos

1. **Setup inicial**: Configurar primeiro workflow (Telegram notifications)
2. **Test webhook**: Implementar webhook endpoint no CrypGo
3. **Iterate**: Expandir workflows baseado no feedback
4. **Monitor**: Acompanhar performance das automações
5. **Scale**: Adicionar workflows mais complexos

---

💡 **Dica**: Comece sempre com workflows simples e vá aumentando a complexidade gradualmente. O N8N permite testar workflows com dados fictícios antes de conectar às APIs reais.

🔔 **Lembre-se**: Sempre configurar error handling e fallbacks nos workflows para evitar falhas em cascata no sistema de trading.