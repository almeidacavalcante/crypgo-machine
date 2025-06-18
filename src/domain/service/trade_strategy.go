package service

import (
	"crypgo-machine/src/domain/entity"
	"fmt"
)

func NewTradeStrategyFactory(strategyType string, params []int) (entity.TradingStrategy, error) {
	switch strategyType {
	case "MovingAverage":
		if len(params) != 2 {
			return nil, fmt.Errorf("MovingAverage expects 2 params")
		}
		return NewMovingAverageStrategy(params[0], params[1]), nil
	case "Breakout":
		if len(params) != 1 {
			return nil, fmt.Errorf("breakout expects 1 param")
		}
		return NewBreakoutStrategy(params[0]), nil
	default:
		return nil, fmt.Errorf("unknown strategy: %s", strategyType)
	}
}
