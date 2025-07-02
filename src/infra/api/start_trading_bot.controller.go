package api

import (
	"crypgo-machine/src/application/usecase"
	"encoding/json"
	"net/http"
)

type StartTradingBotController struct {
	StartTradingBot *usecase.StartTradingBotUseCase
}

func NewStartTradingBotController(startTradingBot *usecase.StartTradingBotUseCase) *StartTradingBotController {
	return &StartTradingBotController{
		StartTradingBot: startTradingBot,
	}
}

func (c *StartTradingBotController) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var rawInput usecase.InputStartTradingBot
	if err := json.NewDecoder(r.Body).Decode(&rawInput); err != nil {
		http.Error(w, "invalid input", http.StatusBadRequest)
		return
	}
	err := c.StartTradingBot.Execute(rawInput)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
}
