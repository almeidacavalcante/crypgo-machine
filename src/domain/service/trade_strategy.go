package service

import (
	"crypgo-machine/src/domain/entity"
	"fmt"
)

type MovingAverageParams struct {
	FastWindow int
	SlowWindow int
}

func NewTradeStrategyFactory(strategyType string, Params interface{}) (entity.TradingStrategy, error) {
	var strategy entity.TradingStrategy

	switch strategyType {
	case "MovingAverage":
		params, ok := Params.(MovingAverageParams)
		if !ok {
			return nil, fmt.Errorf("params must be MovingAverageParams for MovingAverage strategy")
		}
		if params.FastWindow <= 0 || params.SlowWindow <= 0 {
			return nil, fmt.Errorf("missing or invalid fields for MovingAverage: FastWindow and SlowWindow must be > 0")
		}
		strategy = entity.NewMovingAverageStrategy(params.FastWindow, params.SlowWindow)

	default:
		return nil, fmt.Errorf("unknown or invalid strategy: %s", strategyType)
	}

	return strategy, nil
}
