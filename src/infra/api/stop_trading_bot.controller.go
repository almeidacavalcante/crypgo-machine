package api

import (
	"crypgo-machine/src/application/usecase"
	"encoding/json"
	"net/http"
)

type StopTradingBotController struct {
	StopTradingBot *usecase.StopTradingBotUseCase
}

func NewStopTradingBotController(stopTradingBot *usecase.StopTradingBotUseCase) *StopTradingBotController {
	return &StopTradingBotController{
		StopTradingBot: stopTradingBot,
	}
}

func (c *StopTradingBotController) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input usecase.InputStopTradingBot
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := c.StopTradingBot.Execute(input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Trading bot stopped successfully",
		"bot_id":  input.BotId,
	})
}