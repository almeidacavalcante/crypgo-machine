package repository

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/service"
	"crypgo-machine/src/domain/vo"
	"database/sql"
	"fmt"
	"time"
)

type TradeBotRepositoryDatabase struct {
	db *sql.DB
}

func NewTradeBotRepositoryDatabase(db *sql.DB) *TradeBotRepositoryDatabase {
	return &TradeBotRepositoryDatabase{db: db}
}

var _ repository.TradeBotRepository = (*TradeBotRepositoryDatabase)(nil)

func (r *TradeBotRepositoryDatabase) Save(bot *entity.TradeBot) error {
	query := `
		INSERT INTO trade_bots (id, symbol, quantity, strategy_name, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(query,
		string(bot.ID),
		string(bot.Symbol()),
		bot.Quantity(),
		bot.Strategy().Name(),
		string(bot.Status()),
		bot.CreatedAt(),
	)
	return err
}

func (r *TradeBotRepositoryDatabase) Exists(id string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM trade_bots WHERE id=$1)`
	var exists bool
	err := r.db.QueryRow(query, id).Scan(&exists)
	return exists, err
}

func (r *TradeBotRepositoryDatabase) GetTradeByID(id string) (*entity.TradeBot, error) {
	query := `
		SELECT id, symbol, quantity, strategy_name, status, created_at
		FROM trade_bots
		WHERE id = $1
	`

	var (
		botID        string
		symbol       string
		quantity     float64
		strategyName string
		status       string
		createdAt    time.Time
	)

	err := r.db.QueryRow(query, id).Scan(
		&botID,
		&symbol,
		&quantity,
		&strategyName,
		&status,
		&createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var strategy entity.TradingStrategy
	switch strategyName {
	case "MovingAverage":
		strategy = service.NewMovingAverageStrategy(5, 10) // TODO: Use actual parameters
	case "Breakout":
		strategy = service.NewBreakoutStrategy(5) // TODO: Use actual parameters
	default:
		return nil, fmt.Errorf("estratégia desconhecida: %s", strategyName)
	}

	tradeBot := entity.Restore(
		vo.UUID(botID),
		vo.Symbol(symbol),
		quantity,
		strategy,
		entity.Status(status),
		createdAt,
	)

	if err != nil {
		return nil, err
	}

	return tradeBot, nil
}
