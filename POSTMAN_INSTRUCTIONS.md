# 📮 INSTRUÇÕES PARA TESTE NO POSTMAN

## 🎯 **Configuração Campeã para Teste**

### **Endpoint da API:**
```
POST http://localhost:8080/api/v1/trading/create_trading_bot
```

### **Headers:**
```
Content-Type: application/json
```

### **JSON Body (Configuração Campeã):**
```json
{
  "symbol": "SOLBRL",
  "quantity": 0.1,
  "strategy": "MovingAverage",
  "params": {
    "FastWindow": 3,
    "SlowWindow": 10
  },
  "interval_seconds": 1800,
  "initial_capital": 5000.0,
  "trade_amount": 5000.0,
  "currency": "BRL",
  "trading_fees": 0.01,
  "minimum_profit_threshold": 5.0
}
```

---

## 🏆 **EXPLICAÇÃO DOS PARÂMETROS:**

| Campo | Valor | Descrição |
|-------|-------|-----------|
| `symbol` | `"SOLBRL"` | Par de trading (Solana/Real) |
| `quantity` | `0.1` | Quantidade de SOL por trade |
| `strategy` | `"MovingAverage"` | Estratégia de médias móveis |
| `params.FastWindow` | `3` | Janela da média móvel rápida |
| `params.SlowWindow` | `10` | Janela da média móvel lenta |
| `interval_seconds` | `1800` | Intervalo de 30 minutos (1800s) |
| `initial_capital` | `5000.0` | Capital inicial em BRL |
| `trade_amount` | `5000.0` | Valor por trade em BRL |
| `currency` | `"BRL"` | Moeda base para cálculos |
| `trading_fees` | `0.01` | Taxa de trading (0.01%) |
| `minimum_profit_threshold` | `5.0` | Lucro mínimo de 5% para venda |

---

## ⚡ **VARIAÇÕES PARA TESTE:**

### **Configuração Conservadora (Menos Trades):**
```json
{
  "symbol": "SOLBRL",
  "quantity": 0.1,
  "strategy": "MovingAverage",
  "params": {
    "FastWindow": 3,
    "SlowWindow": 10
  },
  "interval_seconds": 1800,
  "initial_capital": 5000.0,
  "trade_amount": 2000.0,
  "currency": "BRL",
  "trading_fees": 0.01,
  "minimum_profit_threshold": 3.0
}
```

### **Configuração Agressiva (Mais Trades):**
```json
{
  "symbol": "SOLBRL",
  "quantity": 0.1,
  "strategy": "MovingAverage",
  "params": {
    "FastWindow": 3,
    "SlowWindow": 10
  },
  "interval_seconds": 1800,
  "initial_capital": 5000.0,
  "trade_amount": 1000.0,
  "currency": "BRL",
  "trading_fees": 0.01,
  "minimum_profit_threshold": 1.0
}
```

### **Configuração com BTCBRL:**
```json
{
  "symbol": "BTCBRL",
  "quantity": 0.001,
  "strategy": "MovingAverage",
  "params": {
    "FastWindow": 3,
    "SlowWindow": 10
  },
  "interval_seconds": 1800,
  "initial_capital": 5000.0,
  "trade_amount": 3000.0,
  "currency": "BRL",
  "trading_fees": 0.01,
  "minimum_profit_threshold": 2.5
}
```

---

## 🚀 **PASSOS PARA TESTAR:**

1. **Inicie o servidor:**
   ```bash
   go run main.go
   ```

2. **Configure o Postman:**
   - Method: `POST`
   - URL: `http://localhost:8080/api/v1/trading/create_trading_bot`
   - Headers: `Content-Type: application/json`
   - Body: Copie um dos JSONs acima

3. **Execute o request**

4. **Resposta esperada:**
   - Status: `201 Created` (sucesso)
   - Body: Vazio em caso de sucesso
   - Status: `400 Bad Request` (erro de validação)

---

## 🔍 **OUTROS ENDPOINTS ÚTEIS:**

### **Listar Trading Bots:**
```
GET http://localhost:8080/api/v1/trading/list
```

### **Iniciar Trading Bot:**
```
POST http://localhost:8080/api/v1/trading/start
```
**Body:**
```json
{
  "bot_id": "uuid-do-bot-criado"
}
```

### **Parar Trading Bot:**
```
POST http://localhost:8080/api/v1/trading/stop
```
**Body:**
```json
{
  "bot_id": "uuid-do-bot-criado"
}
```

### **Backtest (API):**
```
POST http://localhost:8080/api/v1/trading/backtest
```
**Body:**
```json
{
  "symbol": "SOLBRL",
  "strategy": "MovingAverage",
  "fast_window": 3,
  "slow_window": 10,
  "start_date": "2024-05-01",
  "end_date": "2024-05-31",
  "interval": "30m"
}
```

---

## 🎯 **VALIDAÇÕES QUE A API FAZ:**

- ✅ Symbol deve ser válido (SOLBRL, BTCBRL)
- ✅ Quantity deve ser > 0
- ✅ InitialCapital deve ser > 0  
- ✅ TradeAmount deve ser > 0
- ✅ TradingFees deve ser >= 0
- ✅ MinimumProfitThreshold deve ser >= 0
- ✅ FastWindow e SlowWindow devem ser > 0
- ✅ Strategy deve ser "MovingAverage"

---

**💡 Tip:** Use a configuração campeã para ter a maior chance de sucesso baseada nos resultados de otimização!