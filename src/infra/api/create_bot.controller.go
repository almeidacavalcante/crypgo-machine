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
		if err := decodeStrategyParams(rawInput.Params, &m); err != nil {
			http.Error(w, "invalid params for MovingAverage", http.StatusBadRequest)
			return
		}
		params = m
	case "RSI":
		var rsi service.RSIParams
		if err := decodeStrategyParams(rawInput.Params, &rsi); err != nil {
			http.Error(w, "invalid params for RSI", http.StatusBadRequest)
			return
		}
		params = rsi
	default:
		http.Error(w, "unknown strategy", http.StatusBadRequest)
		return
	}

	// Copy the whole decoded input and only replace Params with the typed struct,
	// so fields added to InputCreateTradingBot are never silently dropped.
	input := rawInput
	input.Params = params

	if err := c.CreateTradingBot.Execute(input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// decodeStrategyParams converts the generic JSON params (decoded as a map) into
// the typed params struct expected by the strategy factory.
func decodeStrategyParams(raw interface{}, target interface{}) error {
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}
