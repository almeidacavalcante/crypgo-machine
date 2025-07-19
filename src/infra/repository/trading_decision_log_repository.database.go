package repository

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/vo"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

func (r *TradingDecisionLogRepositoryDatabase) GetRecentLogs(limit int) ([]*entity.TradingDecisionLog, error) {
	query := `
		SELECT id, trading_bot_id, decision, strategy_name, analysis_data, market_data, 
		       current_price, current_possible_profit, timestamp
		FROM trading_decision_logs 
		ORDER BY timestamp DESC 
		LIMIT $1
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRows(rows)
}

func (r *TradingDecisionLogRepositoryDatabase) GetRecentLogsByDecision(decision string, limit int) ([]*entity.TradingDecisionLog, error) {
	query := `
		SELECT id, trading_bot_id, decision, strategy_name, analysis_data, market_data, 
		       current_price, current_possible_profit, timestamp
		FROM trading_decision_logs 
		WHERE decision = $1
		ORDER BY timestamp DESC 
		LIMIT $2
	`

	rows, err := r.db.Query(query, decision, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRows(rows)
}

func (r *TradingDecisionLogRepositoryDatabase) scanRows(rows *sql.Rows) ([]*entity.TradingDecisionLog, error) {
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

		// Create entities
		entityId, err := vo.RestoreEntityId(id)
		if err != nil {
			return nil, err
		}

		tradingBotId, err := vo.RestoreEntityId(botId)
		if err != nil {
			return nil, err
		}

		tradingDecision, err := entity.ParseTradingDecision(decision)
		if err != nil {
			return nil, err
		}

		log := entity.RestoreTradingDecisionLog(
			entityId,
			tradingBotId,
			tradingDecision,
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
// GetLogsWithFilters retrieves logs with optional filters and pagination
func (r *TradingDecisionLogRepositoryDatabase) GetLogsWithFilters(decision string, symbol string, limit int, offset int) ([]*entity.TradingDecisionLog, int, error) {
	// Build WHERE clause dynamically
	var whereConditions []string
	var args []interface{}
	argIndex := 1

	if decision != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("l.decision = $%d", argIndex))
		args = append(args, decision)
		argIndex++
	}

	if symbol != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("b.symbol = $%d", argIndex))
		args = append(args, symbol)
		argIndex++
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// First, get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM trading_decision_logs l
		LEFT JOIN trade_bots b ON l.trading_bot_id = b.id
		%s
	`, whereClause)

	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Then get the actual logs with pagination
	query := fmt.Sprintf(`
		SELECT l.id, l.trading_bot_id, l.decision, l.strategy_name, l.analysis_data, 
			   l.market_data, l.current_price, l.current_possible_profit, l.timestamp
		FROM trading_decision_logs l
		LEFT JOIN trade_bots b ON l.trading_bot_id = b.id
		%s
		ORDER BY l.timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs, err := r.scanRows(rows)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}