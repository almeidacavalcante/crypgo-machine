package api

import (
	"crypgo-machine/src/application/usecase"
	"crypgo-machine/src/domain/entity"
	"encoding/json"
	"net/http"
)

type ListAllTradingBotsController struct {
	ListAllTradingBots *usecase.ListAllTradingBotsUseCase
}

func NewListAllTradingBotsController(listAllTradingBots *usecase.ListAllTradingBotsUseCase) *ListAllTradingBotsController {
	return &ListAllTradingBotsController{
		ListAllTradingBots: listAllTradingBots,
	}
}

func (c *ListAllTradingBotsController) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	bots, err := c.ListAllTradingBots.Execute()

	if err != nil {
		http.Error(w, "failed to list trading bots", http.StatusInternalServerError)
		return
	}

	if len(bots) == 0 {
		http.Error(w, "no trading bots found", http.StatusNotFound)
		return
	}

	var tradingBotsDTOs []entity.TradingBotDTO
	for _, b := range bots {
		tradingBotsDTOs = append(tradingBotsDTOs, b.ToDTO())
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tradingBotsDTOs); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
