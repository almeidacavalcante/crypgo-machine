package repository

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type TradingDecisionLogRepositoryDatabase struct {
	db *sql.DB
}

func NewTradingDecisionLogRepositoryDatabase(db *sql.DB) *TradingDecisionLogRepositoryDatabase {
	return &TradingDecisionLogRepositoryDatabase{db: db}
}

var _ repository.TradingDecisionLogRepository = (*TradingDecisionLogRepositoryDatabase)(nil)

func (r *TradingDecisionLogRepositoryDatabase) Save(log *entity.TradingDecisionLog) error {
	analysisDataJson, err := json.Marshal(log.GetAnalysisData())
	if err != nil {
		return err
	}

	marketDataJson, err := json.Marshal(log.GetMarketData())
	if err != nil {
		return err
	}

	query := `
		INSERT INTO trading_decision_logs (
			id, trading_bot_id, decision, strategy_name, 
			analysis_data, market_data, current_price, current_possible_profit, timestamp
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.db.Exec(query,
		log.GetId().GetValue(),
		log.GetTradingBotId().GetValue(),
		string(log.GetDecision()),
		log.GetStrategyName(),
		string(analysisDataJson),
		string(marketDataJson),
		log.GetCurrentPrice(),
		log.GetCurrentPossibleProfit(),
		log.GetTimestamp(),
	)

	return err
}

func (r *TradingDecisionLogRepositoryDatabase) GetByTradingBotId(tradingBotId string) ([]*entity.TradingDecisionLog, error) {
	return r.GetByTradingBotIdWithLimit(tradingBotId, 0)
}

func (r *TradingDecisionLogRepositoryDatabase) GetByTradingBotIdWithLimit(tradingBotId string, limit int) ([]*entity.TradingDecisionLog, error) {
	query := `
		SELECT id, trading_bot_id, decision, strategy_name, 
			   analysis_data, market_data, current_price, current_possible_profit, timestamp
		FROM trading_decision_logs 
		WHERE trading_bot_id = $1 
		ORDER BY timestamp DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.Query(query, tradingBotId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*entity.TradingDecisionLog
	for rows.Next() {
		var (
			id                   string
			botId                string
			decision             string
			strategyName         string
			analysisDataStr      string
			marketDataStr        string
			currentPrice         float64
			currentPossibleProfit float64
			timestamp            time.Time
		)

		if err := rows.Scan(&id, &botId, &decision, &strategyName,
			&analysisDataStr, &marketDataStr, &currentPrice, &currentPossibleProfit, &timestamp); err != nil {
			return nil, err
		}

		// Deserialize analysis data
		var analysisData map[string]interface{}
		if err := json.Unmarshal([]byte(analysisDataStr), &analysisData); err != nil {
			return nil, err
		}

		// Deserialize market data
		var marketData []vo.Kline
		if err := json.Unmarshal([]byte(marketDataStr), &marketData); err != nil {
			return nil, err
		}

		// Restore entity IDs
		logId, err := vo.RestoreEntityId(id)
		if err != nil {
			return nil, err
		}

		tradingBotEntityId, err := vo.RestoreEntityId(botId)
		if err != nil {
			return nil, err
		}

		log := entity.RestoreTradingDecisionLog(
			logId,
			tradingBotEntityId,
			entity.TradingDecision(decision),
			strategyName,
			analysisData,
			marketData,
			currentPrice,
			currentPossibleProfit,
			timestamp,
		)

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}
