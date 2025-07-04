package api

import (
	"crypgo-machine/src/application/usecase"
	"crypgo-machine/src/domain/service"
	"encoding/json"
	"net/http"
)

type CreateTradingBotController struct {
	CreateTradingBot *usecase.CreateTradingBotUseCase
}

func NewCreateTradingBotController(createTradingBot *usecase.CreateTradingBotUseCase) *CreateTradingBotController {
	return &CreateTradingBotController{
		CreateTradingBot: createTradingBot,
	}
}

func (c *CreateTradingBotController) Handle(w http.ResponseWriter, r *http.Request) {
	var rawInput usecase.InputCreateTradingBot
	if err := json.NewDecoder(r.Body).Decode(&rawInput); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var params interface{}
	switch rawInput.Strategy {
	case "MovingAverage":
		var m service.MovingAverageParams
		// Marshal/Unmarshal: transforma map em struct
		b, _ := json.Marshal(rawInput.Params)
		if err := json.Unmarshal(b, &m); err != nil {
			http.Error(w, "invalid params for MovingAverage", http.StatusBadRequest)
			return
		}
		params = m
	default:
		http.Error(w, "unknown strategy", http.StatusBadRequest)
		return
	}

	input := usecase.InputCreateTradingBot{
		Symbol:                   rawInput.Symbol,
		Quantity:                 rawInput.Quantity,
		Strategy:                 rawInput.Strategy,
		Params:                   params,
		IntervalSeconds:          rawInput.IntervalSeconds,
		InitialCapital:           rawInput.InitialCapital,
		TradeAmount:              rawInput.TradeAmount,
		Currency:                 rawInput.Currency,
		TradingFees:              rawInput.TradingFees,
		MinimumProfitThreshold:   rawInput.MinimumProfitThreshold,
	}

	if err := c.CreateTradingBot.Execute(input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
