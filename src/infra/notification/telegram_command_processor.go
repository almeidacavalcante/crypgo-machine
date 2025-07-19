package notification

import (
	"crypgo-machine/src/application/service"
	"fmt"
	"log"
	"strings"
	"time"
)

type TelegramCommandProcessor struct {
	marketService   *service.MarketSentimentService
	telegramService *TelegramService
}

func NewTelegramCommandProcessor(marketService *service.MarketSentimentService, telegramService *TelegramService) *TelegramCommandProcessor {
	return &TelegramCommandProcessor{
		marketService:   marketService,
		telegramService: telegramService,
	}
}

// ProcessCommand handles command logic and returns response message
func (p *TelegramCommandProcessor) ProcessCommand(command string, args string, chatID int64) string {
	log.Printf("🔄 Processing command: /%s", command)

	switch strings.ToLower(command) {
	case "sentiment":
		return p.handleSentimentCommand(args)
	case "quick":
		return p.handleQuickCommand(args)
	case "status":
		return p.handleStatusCommand(args)
	case "help":
		return p.handleHelpCommand()
	case "oi":
		return p.handleOiCommand()
	default:
		return p.handleUnknownCommand(command)
	}
}

// handleSentimentCommand processes full sentiment analysis
func (p *TelegramCommandProcessor) handleSentimentCommand(args string) string {
	// Note: In future versions, we could send a "processing" message first

	result, err := p.marketService.CollectMarketSentiment()
	if err != nil {
		log.Printf("❌ Error in sentiment analysis: %v", err)
		return fmt.Sprintf("❌ <b>Erro na Análise</b>\n\nNão foi possível realizar a análise de sentiment.\n\n<i>Erro: %s</i>", err.Error())
	}

	// Format comprehensive response
	sentiment := result.Suggestion.GetLevel().String()
	score := result.Suggestion.GetOverallScore().GetValue()
	suggestions := p.marketService.GetSentimentSuggestions(sentiment)

	// Get sentiment emoji and level name
	sentimentEmoji := p.getSentimentEmoji(sentiment)
	sentimentName := p.getSentimentDisplayName(sentiment)

	response := fmt.Sprintf(
		"%s <b>Análise de Sentiment Completa</b>\n\n"+
			"📊 <b>Resultado</b>: %s (%s)\n"+
			"📈 <b>Score</b>: %.3f\n"+
			"🎯 <b>Confiança</b>: %.1f%%\n"+
			"⏰ <b>Horário</b>: %s\n\n"+
			"📋 <b>FONTES DE DADOS</b>:\n"+
			"😨 Fear & Greed Index: %d\n"+
			"📰 News Score: %.3f\n"+
			"🔥 Reddit Score: %.3f\n"+
			"📱 Social Score: %.3f\n\n"+
			"💡 <b>SUGESTÕES CONSULTIVAS</b>:\n"+
			"• <b>Trade Amount</b>: %.1fx multiplier\n"+
			"• <b>Profit Target</b>: %.1f%%\n"+
			"• <b>Interval</b>: %s\n"+
			"• <b>Recomendação</b>: %s\n\n"+
			"📝 <b>Análise</b>:\n%s\n\n"+
			"⚠️ <i>Estas são sugestões consultivas. Revise antes de aplicar.</i>\n\n"+
			"#CrypGo #SentimentAnalysis",
		sentimentEmoji,
		sentimentName,
		sentiment,
		score,
		result.Confidence*100,
		time.Now().Format("15:04:05"),
		result.Sources.GetFearGreedIndex(),
		result.Sources.GetNewsScore(),
		result.Sources.GetRedditScore(),
		result.Sources.GetSocialScore(),
		suggestions.TradeAmountMultiplier,
		suggestions.MinimumProfitThreshold,
		p.formatInterval(suggestions.IntervalSeconds),
		suggestions.Recommendation,
		result.Reasoning,
	)

	log.Printf("✅ Sentiment analysis completed: %s", sentiment)
	return response
}

// handleQuickCommand processes quick sentiment check
func (p *TelegramCommandProcessor) handleQuickCommand(args string) string {
	result, err := p.marketService.QuickSentimentCheck()
	if err != nil {
		log.Printf("❌ Error in quick check: %v", err)
		return fmt.Sprintf("❌ <b>Erro na Verificação</b>\n\nNão foi possível realizar a verificação rápida.\n\n<i>Erro: %s</i>", err.Error())
	}

	sentiment := result.Suggestion.GetLevel().String()
	score := result.Suggestion.GetOverallScore().GetValue()
	sentimentEmoji := p.getSentimentEmoji(sentiment)
	sentimentName := p.getSentimentDisplayName(sentiment)

	response := fmt.Sprintf(
		"⚡ <b>Verificação Rápida</b>\n\n"+
			"%s <b>Sentiment</b>: %s (%s)\n"+
			"📈 <b>Score</b>: %.3f\n"+
			"🎯 <b>Confiança</b>: %.1f%%\n"+
			"⏰ <b>Horário</b>: %s\n\n"+
			"💭 <b>Base</b>: %s\n\n"+
			"💡 <i>Para análise completa, use</i> <code>/sentiment</code>\n\n"+
			"#CrypGo #QuickCheck",
		sentimentEmoji,
		sentimentName,
		sentiment,
		score,
		result.Confidence*100,
		time.Now().Format("15:04:05"),
		result.Reasoning,
	)

	log.Printf("✅ Quick check completed: %s", sentiment)
	return response
}

// handleStatusCommand shows system status
func (p *TelegramCommandProcessor) handleStatusCommand(args string) string {
	// Test data source connectivity
	dataSourceStatus := "✅ Conectado"
	if err := p.marketService.ValidateDataSources(); err != nil {
		dataSourceStatus = fmt.Sprintf("❌ Erro: %s", err.Error())
	}

	response := fmt.Sprintf(
		"📊 <b>Status do Sistema CrypGo</b>\n\n"+
			"🤖 <b>Telegram Bot</b>: ✅ Ativo\n"+
			"📡 <b>Fontes de Dados</b>: %s\n"+
			"🔍 <b>Sentiment Service</b>: ✅ Operacional\n"+
			"⏰ <b>Última Verificação</b>: %s\n\n"+
			"📋 <b>COMANDOS DISPONÍVEIS</b>:\n"+
			"• <code>/sentiment</code> - Análise completa\n"+
			"• <code>/quick</code> - Verificação rápida\n"+
			"• <code>/status</code> - Este status\n"+
			"• <code>/help</code> - Ajuda completa\n\n"+
			"🔗 <b>Acesso Web</b>: http://31.97.249.4/dashboard/\n\n"+
			"#CrypGo #Status",
		dataSourceStatus,
		time.Now().Format("15:04:05"),
	)

	log.Println("✅ Status command completed")
	return response
}

// handleHelpCommand provides comprehensive help
func (p *TelegramCommandProcessor) handleHelpCommand() string {
	response := "❓ <b>Ajuda - CrypGo Telegram Bot</b>\n\n" +
		"🔍 <b>/sentiment</b>\n" +
		"   Análise completa de sentiment do mercado crypto\n" +
		"   • Fear & Greed Index\n" +
		"   • Análise de notícias\n" +
		"   • Sentiment do Reddit\n" +
		"   • Sugestões consultivas\n\n" +
		"⚡ <b>/quick</b>\n" +
		"   Verificação rápida baseada no Fear & Greed Index\n" +
		"   • Mais rápido que análise completa\n" +
		"   • Ideal para monitoramento frequente\n\n" +
		"📊 <b>/status</b>\n" +
		"   Status dos sistemas e conectividade\n" +
		"   • Estado dos serviços\n" +
		"   • Teste de fontes de dados\n\n" +
		"👋 <b>/oi</b>\n" +
		"   Teste de conectividade básico\n\n" +
		"💡 <b>IMPORTANTE</b>:\n" +
		"• Todas as sugestões são consultivas\n" +
		"• Sempre revise antes de aplicar\n" +
		"• Use o dashboard web para configurações\n\n" +
		"🌐 <b>Dashboard</b>: http://31.97.249.4/dashboard/\n\n" +
		"#CrypGo #Help"

	log.Println("✅ Help command completed")
	return response
}

// handleOiCommand provides basic connectivity test
func (p *TelegramCommandProcessor) handleOiCommand() string {
	response := "🤖 <b>OI!</b>\n\n" +
		"CrypGo Bot está funcionando perfeitamente! 🚀\n\n" +
		"📊 <i>Sistema operacional e pronto para análises</i>\n\n" +
		"💡 Use <code>/help</code> para ver todos os comandos disponíveis.\n\n" +
		"#CrypGo #Conectividade"

	log.Println("✅ Oi command completed")
	return response
}

// handleUnknownCommand responds to unrecognized commands
func (p *TelegramCommandProcessor) handleUnknownCommand(command string) string {
	response := fmt.Sprintf(
		"❓ <b>Comando Não Reconhecido</b>\n\n"+
			"O comando <code>/%s</code> não existe.\n\n"+
			"📋 <b>Comandos Disponíveis</b>:\n"+
			"• <code>/sentiment</code> - Análise completa\n"+
			"• <code>/quick</code> - Verificação rápida\n"+
			"• <code>/status</code> - Status do sistema\n"+
			"• <code>/help</code> - Ajuda detalhada\n"+
			"• <code>/oi</code> - Teste de conectividade\n\n"+
			"💡 Use <code>/help</code> para mais informações.\n\n"+
			"#CrypGo #ComandoInválido",
		command,
	)

	log.Printf("⚠️ Unknown command: /%s", command)
	return response
}

// Helper methods

func (p *TelegramCommandProcessor) getSentimentEmoji(sentiment string) string {
	switch sentiment {
	case "very_bullish":
		return "🚀"
	case "bullish":
		return "📈"
	case "neutral":
		return "➡️"
	case "bearish":
		return "📉"
	case "very_bearish":
		return "🔴"
	default:
		return "❓"
	}
}

func (p *TelegramCommandProcessor) getSentimentDisplayName(sentiment string) string {
	switch sentiment {
	case "very_bullish":
		return "Muito Otimista"
	case "bullish":
		return "Otimista"
	case "neutral":
		return "Neutro"
	case "bearish":
		return "Pessimista"
	case "very_bearish":
		return "Muito Pessimista"
	default:
		return "Desconhecido"
	}
}

func (p *TelegramCommandProcessor) formatInterval(seconds int) string {
	switch seconds {
	case 300:
		return "5 min"
	case 600:
		return "10 min"
	case 900:
		return "15 min"
	case 1800:
		return "30 min"
	case 3600:
		return "1 hora"
	default:
		minutes := seconds / 60
		if minutes < 60 {
			return fmt.Sprintf("%d min", minutes)
		}
		hours := minutes / 60
		return fmt.Sprintf("%d hora(s)", hours)
	}
}