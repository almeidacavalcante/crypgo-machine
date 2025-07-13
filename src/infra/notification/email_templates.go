package notification

import (
	"fmt"
	"time"
)

type TradingEventData struct {
	BotID           string    `json:"bot_id"`
	Symbol          string    `json:"symbol"`
	Action          string    `json:"action"` // "BUY" ou "SELL"
	Price           float64   `json:"price"`
	Quantity        float64   `json:"quantity"`
	TotalValue      float64   `json:"total_value"`
	Strategy        string    `json:"strategy"`
	Timestamp       time.Time `json:"timestamp"`
	EntryPrice      float64   `json:"entry_price,omitempty"`      // Para SELL
	ProfitLoss      float64   `json:"profit_loss,omitempty"`      // Para SELL
	ProfitLossPerc  float64   `json:"profit_loss_perc,omitempty"` // Para SELL
	TradingFees     float64   `json:"trading_fees"`
	Currency        string    `json:"currency"`
}

func GenerateBuyEmailTemplate(data TradingEventData) (string, string) {
	subject := fmt.Sprintf("🟢 CrypGo: Compra Executada - %s", data.Symbol)
	
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
		.content { padding: 20px; }
		.operation-details { background-color: #f9f9f9; padding: 15px; border-radius: 5px; margin: 15px 0; }
		.detail-row { display: flex; justify-content: space-between; margin: 8px 0; }
		.label { font-weight: bold; color: #555; }
		.value { color: #333; }
		.buy-action { color: #4CAF50; font-weight: bold; }
		.footer { background-color: #f1f1f1; padding: 15px; text-align: center; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="header">
		<h1>🟢 ORDEM DE COMPRA EXECUTADA</h1>
		<p>Seu trading bot realizou uma compra!</p>
	</div>
	
	<div class="content">
		<div class="operation-details">
			<h3>📊 Detalhes da Operação</h3>
			
			<div class="detail-row">
				<span class="label">Ação:</span>
				<span class="value buy-action">%s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Símbolo:</span>
				<span class="value">%s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Preço de Compra:</span>
				<span class="value">%.2f %s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Quantidade:</span>
				<span class="value">%.6f</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Valor Total:</span>
				<span class="value">%.2f %s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Taxa de Trading:</span>
				<span class="value">%.3f%%</span>
			</div>
		</div>
		
		<div class="operation-details">
			<h3>🤖 Informações do Bot</h3>
			
			<div class="detail-row">
				<span class="label">Bot ID:</span>
				<span class="value">%s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Estratégia:</span>
				<span class="value">%s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Data/Hora:</span>
				<span class="value">%s</span>
			</div>
		</div>
		
		<p><strong>Status:</strong> ✅ Posição aberta com sucesso. O bot agora aguardará o momento ideal para venda baseado na estratégia configurada.</p>
	</div>
	
	<div class="footer">
		<p>CrypGo Trading Bot - Notificação Automática</p>
		<p>Este email foi gerado automaticamente pelo sistema de trading.</p>
	</div>
</body>
</html>`,
		data.Action,
		data.Symbol,
		data.Price, data.Currency,
		data.Quantity,
		data.TotalValue, data.Currency,
		data.TradingFees,
		data.BotID,
		data.Strategy,
		data.Timestamp.Format("02/01/2006 15:04:05"),
	)
	
	return subject, body
}

func GenerateSellEmailTemplate(data TradingEventData) (string, string) {
	profitIcon := "📈"
	profitColor := "#4CAF50"
	if data.ProfitLoss < 0 {
		profitIcon = "📉"
		profitColor = "#f44336"
	}
	
	subject := fmt.Sprintf("🔴 CrypGo: Venda Executada - %s (%.2f%%)", data.Symbol, data.ProfitLossPerc)
	
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
		.header { background-color: #f44336; color: white; padding: 20px; text-align: center; }
		.content { padding: 20px; }
		.operation-details { background-color: #f9f9f9; padding: 15px; border-radius: 5px; margin: 15px 0; }
		.profit-section { background-color: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 5px; margin: 15px 0; }
		.detail-row { display: flex; justify-content: space-between; margin: 8px 0; }
		.label { font-weight: bold; color: #555; }
		.value { color: #333; }
		.sell-action { color: #f44336; font-weight: bold; }
		.profit-value { color: %s; font-weight: bold; font-size: 18px; }
		.footer { background-color: #f1f1f1; padding: 15px; text-align: center; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="header">
		<h1>🔴 ORDEM DE VENDA EXECUTADA</h1>
		<p>Seu trading bot realizou uma venda!</p>
	</div>
	
	<div class="content">
		<div class="operation-details">
			<h3>📊 Detalhes da Operação</h3>
			
			<div class="detail-row">
				<span class="label">Ação:</span>
				<span class="value sell-action">%s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Símbolo:</span>
				<span class="value">%s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Preço de Venda:</span>
				<span class="value">%.2f %s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Quantidade:</span>
				<span class="value">%.6f</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Valor Total:</span>
				<span class="value">%.2f %s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Taxa de Trading:</span>
				<span class="value">%.3f%%</span>
			</div>
		</div>
		
		<div class="profit-section">
			<h3>%s Resultado da Operação</h3>
			
			<div class="detail-row">
				<span class="label">Preço de Entrada:</span>
				<span class="value">%.2f %s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Preço de Saída:</span>
				<span class="value">%.2f %s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Lucro/Prejuízo:</span>
				<span class="value profit-value">%.2f %s (%.2f%%)</span>
			</div>
		</div>
		
		<div class="operation-details">
			<h3>🤖 Informações do Bot</h3>
			
			<div class="detail-row">
				<span class="label">Bot ID:</span>
				<span class="value">%s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Estratégia:</span>
				<span class="value">%s</span>
			</div>
			
			<div class="detail-row">
				<span class="label">Data/Hora:</span>
				<span class="value">%s</span>
			</div>
		</div>
		
		<p><strong>Status:</strong> ✅ Posição fechada com sucesso. O bot agora aguardará uma nova oportunidade de compra.</p>
	</div>
	
	<div class="footer">
		<p>CrypGo Trading Bot - Notificação Automática</p>
		<p>Este email foi gerado automaticamente pelo sistema de trading.</p>
	</div>
</body>
</html>`,
		profitColor,
		data.Action,
		data.Symbol,
		data.Price, data.Currency,
		data.Quantity,
		data.TotalValue, data.Currency,
		data.TradingFees,
		profitIcon,
		data.EntryPrice, data.Currency,
		data.Price, data.Currency,
		data.ProfitLoss, data.Currency, data.ProfitLossPerc,
		data.BotID,
		data.Strategy,
		data.Timestamp.Format("02/01/2006 15:04:05"),
	)
	
	return subject, body
}