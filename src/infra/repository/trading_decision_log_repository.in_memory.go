package repository

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"sort"
)

type TradingDecisionLogRepositoryInMemory struct {
	logs map[string]*entity.TradingDecisionLog
}

func NewTradingDecisionLogRepositoryInMemory() *TradingDecisionLogRepositoryInMemory {
	return &TradingDecisionLogRepositoryInMemory{
		logs: make(map[string]*entity.TradingDecisionLog),
	}
}

var _ repository.TradingDecisionLogRepository = (*TradingDecisionLogRepositoryInMemory)(nil)

func (r *TradingDecisionLogRepositoryInMemory) Save(log *entity.TradingDecisionLog) error {
	r.logs[log.GetId().GetValue()] = log
	return nil
}

func (r *TradingDecisionLogRepositoryInMemory) GetByTradingBotId(tradingBotId string) ([]*entity.TradingDecisionLog, error) {
	return r.GetByTradingBotIdWithLimit(tradingBotId, 0)
}

func (r *TradingDecisionLogRepositoryInMemory) GetByTradingBotIdWithLimit(tradingBotId string, limit int) ([]*entity.TradingDecisionLog, error) {
	var logs []*entity.TradingDecisionLog
	
	for _, log := range r.logs {
		if log.GetTradingBotId().GetValue() == tradingBotId {
			logs = append(logs, log)
		}
	}
	
	// Sort by timestamp DESC (most recent first)
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].GetTimestamp().After(logs[j].GetTimestamp())
	})
	
	// Apply limit if specified
	if limit > 0 && len(logs) > limit {
		logs = logs[:limit]
	}
	
	return logs, nil
}

func (r *TradingDecisionLogRepositoryInMemory) Clear() {
	r.logs = make(map[string]*entity.TradingDecisionLog)
}