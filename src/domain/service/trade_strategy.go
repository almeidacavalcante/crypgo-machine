package service

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"fmt"
)

type MovingAverageParams struct {
	FastWindow        int
	SlowWindow        int
	StoplossThreshold float64
}

type RSIParams struct {
	Period              int
	OversoldThreshold   float64
	OverboughtThreshold float64
	StoplossThreshold   float64
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
		
		minimumSpread, _ := vo.NewMinimumSpread(0.1)
		
		// Create with stoploss if provided, otherwise use default constructor
		if params.StoplossThreshold > 0 {
			strategy = entity.NewMovingAverageStrategyWithStoploss(params.FastWindow, params.SlowWindow, minimumSpread, params.StoplossThreshold)
		} else {
			strategy = entity.NewMovingAverageStrategy(params.FastWindow, params.SlowWindow)
		}

	case "RSI":
		params, ok := Params.(RSIParams)
		if !ok {
			return nil, fmt.Errorf("params must be RSIParams for RSI strategy")
		}
		if params.Period <= 0 {
			return nil, fmt.Errorf("missing or invalid fields for RSI: Period must be > 0")
		}
		if params.OversoldThreshold <= 0 || params.OversoldThreshold >= 100 {
			return nil, fmt.Errorf("OversoldThreshold must be between 0 and 100")
		}
		if params.OverboughtThreshold <= 0 || params.OverboughtThreshold >= 100 {
			return nil, fmt.Errorf("OverboughtThreshold must be between 0 and 100")
		}
		if params.OversoldThreshold >= params.OverboughtThreshold {
			return nil, fmt.Errorf("OversoldThreshold must be less than OverboughtThreshold")
		}
		
		// Use default thresholds if not provided
		oversold := params.OversoldThreshold
		overbought := params.OverboughtThreshold
		if oversold == 0 {
			oversold = 30.0
		}
		if overbought == 0 {
			overbought = 70.0
		}
		
		minimumSpread, _ := vo.NewMinimumSpread(0.1)
		
		// Create with stoploss if provided, otherwise use custom thresholds or defaults
		if params.StoplossThreshold > 0 {
			strategy = entity.NewRSIStrategyWithStoploss(params.Period, oversold, overbought, minimumSpread, params.StoplossThreshold)
		} else if oversold != 30.0 || overbought != 70.0 {
			strategy = entity.NewRSIStrategyWithCustomThresholds(params.Period, oversold, overbought, minimumSpread)
		} else {
			strategy = entity.NewRSIStrategy(params.Period)
		}

	default:
		return nil, fmt.Errorf("unknown or invalid strategy: %s", strategyType)
	}

	return strategy, nil
}
