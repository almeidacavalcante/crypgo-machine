package usecase

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/service"
	"crypgo-machine/src/domain/vo"
	"fmt"
	"github.com/adshao/go-binance/v2"
)

type MovingAverageParams struct {
	FastWindow int
	SlowWindow int
}

type BreakoutParams struct {
	Lookback int
}

type CreateTradingBotUseCase struct {
	tradeBotRepository repository.TradeBotRepository
	client             binance.Client
}

func NewCreateTradingBotUseCase(tradeBotRepository repository.TradeBotRepository, client binance.Client) *CreateTradingBotUseCase {
	return &CreateTradingBotUseCase{
		tradeBotRepository: tradeBotRepository,
		client:             client,
	}
}

type Input struct {
	Symbol   string
	Quantity float64
	Strategy string
	Params   interface{} // Ou json.RawMessage, ou map[string]interface{}, etc.
}

func (uc *CreateTradingBotUseCase) Execute(input Input) error {
	symbol, err := vo.NewSymbol(input.Symbol)
	if err != nil {
		return fmt.Errorf("invalid symbol: %s", err)
	}
	if input.Quantity <= 0 {
		return fmt.Errorf("invalid quantity: must be greater than zero")
	}

	var strategy entity.TradingStrategy

	switch input.Strategy {
	case "MovingAverage":
		params, ok := input.Params.(MovingAverageParams)
		if !ok {
			return fmt.Errorf("params must be MovingAverageParams for MovingAverage strategy")
		}
		// >>> Validação dos campos obrigatórios
		if params.FastWindow <= 0 || params.SlowWindow <= 0 {
			return fmt.Errorf("missing or invalid fields for MovingAverage: FastWindow and SlowWindow must be > 0")
		}
		strategy = service.NewMovingAverageStrategy(params.FastWindow, params.SlowWindow)

	case "Breakout":
		params, ok := input.Params.(BreakoutParams)
		if !ok {
			return fmt.Errorf("params must be BreakoutParams for Breakout strategy")
		}
		if params.Lookback <= 0 {
			return fmt.Errorf("missing or invalid fields for Breakout: Lookback must be > 0")
		}
		strategy = service.NewBreakoutStrategy(params.Lookback)

	default:
		return fmt.Errorf("unknown or invalid strategy: %s", input.Strategy)
	}

	bot := entity.NewTradeBot(
		vo.NewUUID(),
		symbol,
		input.Quantity,
		strategy,
	)

	return uc.tradeBotRepository.Save(bot)
}
