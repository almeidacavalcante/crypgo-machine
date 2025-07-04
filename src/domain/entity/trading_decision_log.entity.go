package entity

import (
	"crypgo-machine/src/domain/vo"
	"time"
)

type TradingDecisionLog struct {
	Id                   *vo.EntityId
	tradingBotId         *vo.EntityId
	decision             TradingDecision
	strategyName         string
	analysisData         map[string]interface{} // fast, slow, etc.
	marketData           []vo.Kline
	currentPrice         float64
	currentPossibleProfit float64 // Potential profit if position were closed now
	timestamp            time.Time
}

func NewTradingDecisionLog(
	tradingBotId *vo.EntityId,
	decision TradingDecision,
	strategyName string,
	analysisData map[string]interface{},
	marketData []vo.Kline,
	currentPrice float64,
	currentPossibleProfit float64,
) *TradingDecisionLog {
	return &TradingDecisionLog{
		Id:                   vo.NewEntityId(),
		tradingBotId:         tradingBotId,
		decision:             decision,
		strategyName:         strategyName,
		analysisData:         analysisData,
		marketData:           marketData,
		currentPrice:         currentPrice,
		currentPossibleProfit: currentPossibleProfit,
		timestamp:            time.Now(),
	}
}

func RestoreTradingDecisionLog(
	id *vo.EntityId,
	tradingBotId *vo.EntityId,
	decision TradingDecision,
	strategyName string,
	analysisData map[string]interface{},
	marketData []vo.Kline,
	currentPrice float64,
	currentPossibleProfit float64,
	timestamp time.Time,
) *TradingDecisionLog {
	return &TradingDecisionLog{
		Id:                   id,
		tradingBotId:         tradingBotId,
		decision:             decision,
		strategyName:         strategyName,
		analysisData:         analysisData,
		marketData:           marketData,
		currentPrice:         currentPrice,
		currentPossibleProfit: currentPossibleProfit,
		timestamp:            timestamp,
	}
}

func (t *TradingDecisionLog) GetId() *vo.EntityId {
	return t.Id
}

func (t *TradingDecisionLog) GetTradingBotId() *vo.EntityId {
	return t.tradingBotId
}

func (t *TradingDecisionLog) GetDecision() TradingDecision {
	return t.decision
}

func (t *TradingDecisionLog) GetStrategyName() string {
	return t.strategyName
}

func (t *TradingDecisionLog) GetAnalysisData() map[string]interface{} {
	return t.analysisData
}

func (t *TradingDecisionLog) GetMarketData() []vo.Kline {
	return t.marketData
}

func (t *TradingDecisionLog) GetCurrentPrice() float64 {
	return t.currentPrice
}

func (t *TradingDecisionLog) GetTimestamp() time.Time {
	return t.timestamp
}

func (t *TradingDecisionLog) GetCurrentPossibleProfit() float64 {
	return t.currentPossibleProfit
}