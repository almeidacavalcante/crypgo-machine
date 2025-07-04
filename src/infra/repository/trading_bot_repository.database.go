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

type TradingBotRepositoryDatabase struct {
	db *sql.DB
}

func NewTradingBotRepositoryDatabase(db *sql.DB) *TradingBotRepositoryDatabase {
	return &TradingBotRepositoryDatabase{db: db}
}

var _ repository.TradingBotRepository = (*TradingBotRepositoryDatabase)(nil)

func (r *TradingBotRepositoryDatabase) Save(bot *entity.TradingBot) error {
	strategyParams, err := json.Marshal(bot.GetStrategy().GetParams())
	if err != nil {
		return err
	}

	query := `
		INSERT INTO trade_bots (id, symbol, quantity, strategy_name, strategy_params, status, is_positioned, interval_seconds, initial_capital, trade_amount, currency, trading_fees, minimum_profit_threshold, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err = r.db.Exec(query,
		string(bot.Id.GetValue()),
		string(bot.GetSymbol().GetValue()),
		bot.GetQuantity(),
		bot.GetStrategy().GetName(),
		string(strategyParams),
		string(bot.GetStatus()),
		bot.GetIsPositioned(),
		bot.GetIntervalSeconds(),
		bot.GetInitialCapital(),
		bot.GetTradeAmount(),
		bot.GetCurrency(),
		bot.GetTradingFees(),
		bot.GetMinimumProfitThreshold(),
		bot.GetCreatedAt(),
	)
	return err
}

func (r *TradingBotRepositoryDatabase) Update(bot *entity.TradingBot) error {
	strategyParams, err := json.Marshal(bot.GetStrategy().GetParams())
	if err != nil {
		return err
	}

	query := `
		UPDATE trade_bots
		SET symbol = $2, quantity = $3, strategy_name = $4, strategy_params = $5, status = $6, is_positioned = $7, interval_seconds = $8, initial_capital = $9, trade_amount = $10, currency = $11, trading_fees = $12, minimum_profit_threshold = $13, created_at = $14
		WHERE id = $1
	`
	_, err = r.db.Exec(query,
		string(bot.Id.GetValue()),
		string(bot.GetSymbol().GetValue()),
		bot.GetQuantity(),
		bot.GetStrategy().GetName(),
		string(strategyParams),
		string(bot.GetStatus()),
		bot.GetIsPositioned(),
		bot.GetIntervalSeconds(),
		bot.GetInitialCapital(),
		bot.GetTradeAmount(),
		bot.GetCurrency(),
		bot.GetTradingFees(),
		bot.GetMinimumProfitThreshold(),
		bot.GetCreatedAt(),
	)
	return err
}

func (r *TradingBotRepositoryDatabase) Exists(id string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM trade_bots WHERE id=$1)`
	var exists bool
	err := r.db.QueryRow(query, id).Scan(&exists)
	return exists, err
}

func (r *TradingBotRepositoryDatabase) GetTradeByID(id string) (*entity.TradingBot, error) {
	query := `
		SELECT id, symbol, quantity, strategy_name, strategy_params, status, is_positioned, interval_seconds, initial_capital, trade_amount, currency, trading_fees, minimum_profit_threshold, created_at
		FROM trade_bots
		WHERE id = $1
	`

	var (
		botId                  string
		symbol                 string
		quantity               float64
		strategyName           string
		strategyParams         string
		status                 string
		isPositioned           bool
		intervalSeconds        int
		initialCapital         float64
		tradeAmount            float64
		currency               string
		tradingFees            float64
		minimumProfitThreshold float64
		createdAt              time.Time
	)

	err := r.db.QueryRow(query, id).Scan(
		&botId,
		&symbol,
		&quantity,
		&strategyName,
		&strategyParams,
		&status,
		&isPositioned,
		&intervalSeconds,
		&initialCapital,
		&tradeAmount,
		&currency,
		&tradingFees,
		&minimumProfitThreshold,
		&createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	strategy, err := r.buildStrategyFromParams(strategyName, strategyParams)
	if err != nil {
		return nil, err
	}

	symbolInstance, _ := vo.NewSymbol(symbol)
	restoredId, _ := vo.RestoreEntityId(botId)
	tradeBot := entity.Restore(
		restoredId,
		symbolInstance,
		quantity,
		strategy,
		entity.Status(status),
		isPositioned,
		intervalSeconds,
		initialCapital,
		tradeAmount,
		currency,
		tradingFees,
		minimumProfitThreshold,
		createdAt,
	)

	return tradeBot, nil
}

func (r *TradingBotRepositoryDatabase) buildStrategyFromParams(strategyName, strategyParams string) (entity.TradingStrategy, error) {
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(strategyParams), &params); err != nil {
		return nil, err
	}

	switch strategyName {
	case "MovingAverage":
		fast := int(params["FastWindow"].(float64))
		slow := int(params["SlowWindow"].(float64))
		return entity.NewMovingAverageStrategy(fast, slow), nil
	default:
		return nil, fmt.Errorf("estratégia desconhecida: %s", strategyName)
	}
}

func (r *TradingBotRepositoryDatabase) GetAllTradingBots() ([]*entity.TradingBot, error) {
	query := `
		SELECT id, symbol, quantity, strategy_name, strategy_params, status, is_positioned, interval_seconds, initial_capital, trade_amount, currency, trading_fees, minimum_profit_threshold, created_at
		FROM trade_bots
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bots []*entity.TradingBot
	for rows.Next() {
		var (
			botID                  string
			symbol                 string
			quantity               float64
			strategyName           string
			strategyParams         string
			status                 string
			isPositioned           bool
			intervalSeconds        int
			initialCapital         float64
			tradeAmount            float64
			currency               string
			tradingFees            float64
			minimumProfitThreshold float64
			createdAt              time.Time
		)
		if err := rows.Scan(&botID, &symbol, &quantity, &strategyName, &strategyParams, &status, &isPositioned, &intervalSeconds, &initialCapital, &tradeAmount, &currency, &tradingFees, &minimumProfitThreshold, &createdAt); err != nil {
			return nil, err
		}

		strategy, err := r.buildStrategyFromParams(strategyName, strategyParams)
		if err != nil {
			return nil, err
		}

		symbolInstance, errSymbol := vo.NewSymbol(symbol)
		if errSymbol != nil {
			return nil, errSymbol
		}

		restoredId, errRestoredId := vo.RestoreEntityId(botID)
		if errRestoredId != nil {
			return nil, errRestoredId
		}
		bot := entity.Restore(
			restoredId,
			symbolInstance,
			quantity,
			strategy,
			entity.Status(status),
			isPositioned,
			intervalSeconds,
			initialCapital,
			tradeAmount,
			currency,
			tradingFees,
			minimumProfitThreshold,
			createdAt,
		)

		bots = append(bots, bot)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return bots, nil
}
