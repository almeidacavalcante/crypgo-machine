package api

import (
	"crypgo-machine/src/application/usecase"
	"encoding/json"
	"net/http"
)

type CreateTradingBotController struct {
	CreateTradingBotUseCase *usecase.CreateTradingBotUseCase
}

func NewCreateTradingBotController(createTradingBotUseCase *usecase.CreateTradingBotUseCase) *CreateTradingBotController {
	return &CreateTradingBotController{
		CreateTradingBotUseCase: createTradingBotUseCase,
	}
}

func (c *CreateTradingBotController) CreateBot(w http.ResponseWriter, r *http.Request) {
	var rawInput usecase.Input
	if err := json.NewDecoder(r.Body).Decode(&rawInput); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var params interface{}
	switch rawInput.Strategy {
	case "MovingAverage":
		var m usecase.MovingAverageParams
		// Marshal/Unmarshal: transforma map em struct
		b, _ := json.Marshal(rawInput.Params)
		if err := json.Unmarshal(b, &m); err != nil {
			http.Error(w, "invalid params for MovingAverage", http.StatusBadRequest)
			return
		}
		params = m
	case "Breakout":
		var b usecase.BreakoutParams
		bs, _ := json.Marshal(rawInput.Params)
		if err := json.Unmarshal(bs, &b); err != nil {
			http.Error(w, "invalid params for Breakout", http.StatusBadRequest)
			return
		}
		params = b
	default:
		http.Error(w, "unknown strategy", http.StatusBadRequest)
		return
	}

	input := usecase.Input{
		Symbol:   rawInput.Symbol,
		Quantity: rawInput.Quantity,
		Strategy: rawInput.Strategy,
		Params:   params,
	}

	if err := c.CreateTradingBotUseCase.Execute(input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
