package repository

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"sort"
	"sync"
)

type TradingDecisionLogRepositoryInMemory struct {
	logs map[string]*entity.TradingDecisionLog
	mu   sync.RWMutex
}

func (r *TradingDecisionLogRepositoryInMemory) GetLogsWithFilters(decision string, symbol string, limit int, offset int) ([]*entity.TradingDecisionLog, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Filter all logs
	var filtered []*entity.TradingDecisionLog
	for _, log := range r.logs {
		// Filter by decision if specified
		if decision != "" && string(log.GetDecision()) != decision {
			continue
		}

		// TODO: Filter by symbol requires bot information
		// For now, skip symbol filtering in in-memory implementation
		if symbol != "" {
			// Skip symbol filtering for in-memory repo
		}

		filtered = append(filtered, log)
	}

	// Sort by timestamp descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].GetTimestamp().After(filtered[j].GetTimestamp())
	})

	total := len(filtered)

	// Apply pagination
	start := offset
	if start > len(filtered) {
		return []*entity.TradingDecisionLog{}, total, nil
	}

	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

func NewTradingDecisionLogRepositoryInMemory() *TradingDecisionLogRepositoryInMemory {
	return &TradingDecisionLogRepositoryInMemory{
		logs: make(map[string]*entity.TradingDecisionLog),
	}
}

var _ repository.TradingDecisionLogRepository = (*TradingDecisionLogRepositoryInMemory)(nil)

func (r *TradingDecisionLogRepositoryInMemory) Save(log *entity.TradingDecisionLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs[log.GetId().GetValue()] = log
	return nil
}

func (r *TradingDecisionLogRepositoryInMemory) GetByTradingBotId(tradingBotId string) ([]*entity.TradingDecisionLog, error) {
	return r.GetByTradingBotIdWithLimit(tradingBotId, 0)
}

func (r *TradingDecisionLogRepositoryInMemory) GetByTradingBotIdWithLimit(tradingBotId string, limit int) ([]*entity.TradingDecisionLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

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

func (r *TradingDecisionLogRepositoryInMemory) GetRecentLogs(limit int) ([]*entity.TradingDecisionLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert map to slice
	var logs []*entity.TradingDecisionLog
	for _, log := range r.logs {
		logs = append(logs, log)
	}

	// Sort by timestamp descending
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].GetTimestamp().After(logs[j].GetTimestamp())
	})

	if limit > 0 && limit < len(logs) {
		logs = logs[:limit]
	}

	return logs, nil
}

func (r *TradingDecisionLogRepositoryInMemory) GetRecentLogsByDecision(decision string, limit int) ([]*entity.TradingDecisionLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Filter by decision and sort by timestamp descending
	var filtered []*entity.TradingDecisionLog
	for _, log := range r.logs {
		if string(log.GetDecision()) == decision {
			filtered = append(filtered, log)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].GetTimestamp().After(filtered[j].GetTimestamp())
	})

	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

func (r *TradingDecisionLogRepositoryInMemory) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = make(map[string]*entity.TradingDecisionLog)
}
