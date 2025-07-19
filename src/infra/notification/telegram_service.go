package notification

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramService struct {
	bot     *tgbotapi.BotAPI
	chatID  int64
	enabled bool
}

type TelegramData struct {
	ChatID  int64
	Message string
}

func NewTelegramService() *TelegramService {
	botToken := getEnvOrDefaultTelegram("TELEGRAM_BOT_TOKEN", "")
	chatIDStr := getEnvOrDefaultTelegram("TELEGRAM_CHAT_ID", "")
	
	if botToken == "" {
		fmt.Println("⚠️ TELEGRAM_BOT_TOKEN not configured, Telegram notifications disabled")
		return &TelegramService{enabled: false}
	}
	
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		fmt.Printf("❌ Error creating Telegram bot: %v\n", err)
		return &TelegramService{enabled: false}
	}
	
	// Parse chat ID
	var chatID int64
	if chatIDStr != "" {
		if id, err := strconv.ParseInt(chatIDStr, 10, 64); err == nil {
			chatID = id
		} else {
			fmt.Printf("⚠️ Invalid TELEGRAM_CHAT_ID: %s\n", chatIDStr)
		}
	}
	
	// Test bot connection
	me, err := bot.GetMe()
	if err != nil {
		fmt.Printf("❌ Error testing Telegram bot connection: %v\n", err)
		return &TelegramService{enabled: false}
	}
	
	fmt.Printf("✅ Telegram bot connected successfully: @%s\n", me.UserName)
	
	return &TelegramService{
		bot:     bot,
		chatID:  chatID,
		enabled: true,
	}
}

func (t *TelegramService) SendMessage(data TelegramData) error {
	if !t.enabled {
		fmt.Printf("⚠️ Telegram not configured, simulating message send:\n")
		fmt.Printf("📱 Chat ID: %d\n", data.ChatID)
		fmt.Printf("📱 Message: %s\n", data.Message)
		fmt.Printf("---\n")
		return nil
	}
	
	// Use provided chat ID or default
	chatID := data.ChatID
	if chatID == 0 {
		chatID = t.chatID
	}
	
	if chatID == 0 {
		return fmt.Errorf("no chat ID specified and no default chat ID configured")
	}
	
	msg := tgbotapi.NewMessage(chatID, data.Message)
	msg.ParseMode = "HTML" // Enable HTML formatting
	
	_, err := t.bot.Send(msg)
	if err != nil {
		fmt.Printf("❌ Error sending Telegram message: %v\n", err)
		return err
	}
	
	fmt.Printf("✅ Telegram message sent successfully to chat %d\n", chatID)
	return nil
}

// SendSimpleMessage sends a message to the default chat ID
func (t *TelegramService) SendSimpleMessage(message string) error {
	return t.SendMessage(TelegramData{
		ChatID:  t.chatID,
		Message: message,
	})
}

// SendOi sends a simple "OI" message - MVP functionality
func (t *TelegramService) SendOi() error {
	message := "🤖 <b>OI!</b>\n\nCrypGo Bot está funcionando! 🚀"
	return t.SendSimpleMessage(message)
}

// IsEnabled returns whether Telegram service is properly configured
func (t *TelegramService) IsEnabled() bool {
	return t.enabled
}

// GetBotInfo returns bot information if available
func (t *TelegramService) GetBotInfo() (string, error) {
	if !t.enabled {
		return "", fmt.Errorf("telegram service not enabled")
	}
	
	me, err := t.bot.GetMe()
	if err != nil {
		return "", err
	}
	
	return fmt.Sprintf("Bot: @%s (ID: %d)", me.UserName, me.ID), nil
}

// Helper function already exists in email_service.go, but we need it here too
func getEnvOrDefaultTelegram(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return strings.TrimSpace(value)
	}
	return defaultValue
}