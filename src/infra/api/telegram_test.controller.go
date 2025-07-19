package api

import (
	"encoding/json"
	"net/http"

	"crypgo-machine/src/infra/notification"
)

type TelegramTestController struct {
	telegramService *notification.TelegramService
}

type TelegramTestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	BotInfo string `json:"bot_info,omitempty"`
	Error   string `json:"error,omitempty"`
}

func NewTelegramTestController(telegramService *notification.TelegramService) *TelegramTestController {
	return &TelegramTestController{
		telegramService: telegramService,
	}
}

// GET /api/v1/telegram/test - Send "OI" message
func (t *TelegramTestController) SendOi(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if !t.telegramService.IsEnabled() {
		response := TelegramTestResponse{
			Success: false,
			Message: "Telegram service not configured",
			Error:   "TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID not set",
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Send OI message
	err := t.telegramService.SendOi()
	if err != nil {
		response := TelegramTestResponse{
			Success: false,
			Message: "Failed to send Telegram message",
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get bot info
	botInfo, _ := t.telegramService.GetBotInfo()

	response := TelegramTestResponse{
		Success: true,
		Message: "OI message sent successfully via Telegram!",
		BotInfo: botInfo,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GET /api/v1/telegram/status - Check Telegram service status
func (t *TelegramTestController) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := TelegramTestResponse{
		Success: t.telegramService.IsEnabled(),
		Message: "Telegram service status",
	}

	if t.telegramService.IsEnabled() {
		botInfo, err := t.telegramService.GetBotInfo()
		if err != nil {
			response.Error = err.Error()
		} else {
			response.BotInfo = botInfo
		}
	} else {
		response.Message = "Telegram service not configured"
		response.Error = "TELEGRAM_BOT_TOKEN not set or invalid"
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}