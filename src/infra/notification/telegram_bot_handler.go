package notification

import (
	"crypgo-machine/src/application/service"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramBotHandler struct {
	bot                *tgbotapi.BotAPI
	authorizedChatID   int64
	marketService      *service.MarketSentimentService
	commandProcessor   *TelegramCommandProcessor
	enabled            bool
	stopChannel        chan bool
	running            bool
}

func NewTelegramBotHandler(telegramService *TelegramService, marketService *service.MarketSentimentService) *TelegramBotHandler {
	if !telegramService.IsEnabled() {
		log.Println("⚠️ Telegram service not enabled, bot handler disabled")
		return &TelegramBotHandler{enabled: false}
	}

	// Extract bot and chatID from TelegramService
	bot := telegramService.bot
	chatID := telegramService.chatID

	if bot == nil {
		log.Println("❌ No Telegram bot available for handler")
		return &TelegramBotHandler{enabled: false}
	}

	commandProcessor := NewTelegramCommandProcessor(marketService, telegramService)

	return &TelegramBotHandler{
		bot:                bot,
		authorizedChatID:   chatID,
		marketService:      marketService,
		commandProcessor:   commandProcessor,
		enabled:            true,
		stopChannel:        make(chan bool),
		running:            false,
	}
}

// Start begins processing Telegram updates
func (h *TelegramBotHandler) Start() error {
	if !h.enabled {
		log.Println("⚠️ Telegram bot handler not enabled, skipping start")
		return nil
	}

	if h.running {
		return fmt.Errorf("telegram bot handler is already running")
	}

	log.Println("🤖 Starting Telegram Bot Handler...")

	// Configure update settings
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := h.bot.GetUpdatesChan(u)

	h.running = true
	log.Println("✅ Telegram Bot Handler started successfully")

	go h.processUpdates(updates)

	return nil
}

// Stop halts the bot handler
func (h *TelegramBotHandler) Stop() {
	if !h.running {
		return
	}

	log.Println("⏹️ Stopping Telegram Bot Handler")
	h.running = false
	close(h.stopChannel)
	h.bot.StopReceivingUpdates()
	log.Println("✅ Telegram Bot Handler stopped")
}

// processUpdates handles incoming messages
func (h *TelegramBotHandler) processUpdates(updates tgbotapi.UpdatesChannel) {
	for {
		select {
		case update := <-updates:
			if update.Message != nil {
				h.handleMessage(update.Message)
			}
		case <-h.stopChannel:
			log.Println("📨 Update processor stopped")
			return
		}
	}
}

// handleMessage processes individual messages
func (h *TelegramBotHandler) handleMessage(message *tgbotapi.Message) {
	// Log all incoming messages
	log.Printf("📨 Received message from %d: %s", message.Chat.ID, message.Text)

	// Check if message is from authorized chat
	if message.Chat.ID != h.authorizedChatID {
		log.Printf("⚠️ Unauthorized chat attempt from %d", message.Chat.ID)
		h.sendUnauthorizedMessage(message.Chat.ID)
		return
	}

	// Check if message is a command
	if !message.IsCommand() {
		h.sendHelpMessage(message.Chat.ID)
		return
	}

	// Process command
	command := message.Command()
	args := message.CommandArguments()

	log.Printf("🔄 Processing command: /%s with args: %s", command, args)

	// Send immediate processing message for sentiment analysis (can take time)
	if command == "sentiment" {
		processingMsg := "🧠 <b>Analisando Sentimento do Mercado</b>\n\n" +
			"⏳ Coletando dados de múltiplas fontes...\n" +
			"📊 Processando com IA avançada...\n\n" +
			"<i>⏱️ Isso pode levar até 1 minuto. Aguarde...</i>"
		h.sendMessage(message.Chat.ID, processingMsg)
	}

	response := h.commandProcessor.ProcessCommand(command, args, message.Chat.ID)
	h.sendMessage(message.Chat.ID, response)
}

// sendMessage sends a message to a specific chat
func (h *TelegramBotHandler) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	_, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("❌ Error sending message to chat %d: %v", chatID, err)
	} else {
		log.Printf("✅ Message sent to chat %d", chatID)
	}
}

// sendUnauthorizedMessage responds to unauthorized users
func (h *TelegramBotHandler) sendUnauthorizedMessage(chatID int64) {
	message := "🚫 <b>Acesso Negado</b>\n\nEste bot é privado e apenas usuários autorizados podem usá-lo."
	h.sendMessage(chatID, message)
}

// sendHelpMessage responds to non-command messages
func (h *TelegramBotHandler) sendHelpMessage(chatID int64) {
	message := "ℹ️ <b>Comandos Disponíveis</b>\n\n" +
		"🔍 <code>/sentiment</code> - Análise completa de mercado\n" +
		"⚡ <code>/quick</code> - Verificação rápida (Fear & Greed)\n" +
		"📊 <code>/status</code> - Status dos bots e sistema\n" +
		"❓ <code>/help</code> - Esta mensagem de ajuda\n" +
		"👋 <code>/oi</code> - Teste de conectividade\n\n" +
		"💡 <i>Digite um comando para começar!</i>"
	h.sendMessage(chatID, message)
}

// IsRunning returns whether the handler is currently processing updates
func (h *TelegramBotHandler) IsRunning() bool {
	return h.running
}

// IsEnabled returns whether the handler is properly configured
func (h *TelegramBotHandler) IsEnabled() bool {
	return h.enabled
}

// GetBotInfo returns information about the bot
func (h *TelegramBotHandler) GetBotInfo() string {
	if !h.enabled || h.bot == nil {
		return "Bot handler not enabled"
	}

	me, err := h.bot.GetMe()
	if err != nil {
		return fmt.Sprintf("Error getting bot info: %v", err)
	}

	return fmt.Sprintf("Bot: @%s (ID: %d), Chat: %d, Running: %t",
		me.UserName, me.ID, h.authorizedChatID, h.running)
}

// SendDirectMessage allows external services to send messages via the bot
func (h *TelegramBotHandler) SendDirectMessage(message string) error {
	if !h.enabled {
		return fmt.Errorf("telegram bot handler not enabled")
	}

	h.sendMessage(h.authorizedChatID, message)
	return nil
}