package usecase

import (
	"crypgo-machine/src/application/repository"
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/domain/service"
	"crypgo-machine/src/domain/vo"
	"crypgo-machine/src/infra/queue"
	"encoding/json"
	"fmt"
	"github.com/adshao/go-binance/v2"
)

type CreateTradingBotUseCase struct {
	tradingBotRepository repository.TradingBotRepository
	client               binance.Client
	messageBroker        queue.MessageBroker
	exchangeName         string
}

func NewCreateTradingBotUseCase(
	tradingBotRepository repository.TradingBotRepository,
	client binance.Client,
	messageBroker queue.MessageBroker,
	exchangeName string,
) *CreateTradingBotUseCase {
	return &CreateTradingBotUseCase{
		tradingBotRepository: tradingBotRepository,
		client:               client,
		messageBroker:        messageBroker,
		exchangeName:         exchangeName,
	}
}

type InputCreateTradingBot struct {
	Symbol                   string      `json:"symbol"`
	Quantity                 float64     `json:"quantity"`
	Strategy                 string      `json:"strategy"`
	Params                   interface{} `json:"params"`
	IntervalSeconds          int         `json:"interval_seconds"`
	InitialCapital           float64     `json:"initial_capital"`
	TradeAmount              float64     `json:"trade_amount"`
	Currency                 string      `json:"currency"`
	TradingFees              float64     `json:"trading_fees"`
	MinimumProfitThreshold   float64     `json:"minimum_profit_threshold"`
	UseFixedQuantity         bool        `json:"use_fixed_quantity"`
}

func (uc *CreateTradingBotUseCase) Execute(input InputCreateTradingBot) error {
	symbol, err := vo.NewSymbol(input.Symbol)
	if err != nil {
		return fmt.Errorf("invalid symbol: %s", err)
	}
	if input.Quantity <= 0 {
		return fmt.Errorf("invalid quantity: must be greater than zero")
	}
	if input.InitialCapital <= 0 {
		return fmt.Errorf("invalid initial capital: must be greater than zero")
	}
	if input.TradeAmount <= 0 {
		return fmt.Errorf("invalid trade amount: must be greater than zero")
	}
	if input.TradingFees < 0 {
		return fmt.Errorf("invalid trading fees: must be greater than or equal to zero")
	}
	if input.MinimumProfitThreshold < 0 {
		return fmt.Errorf("invalid minimum profit threshold: must be greater than or equal to zero")
	}

	strategy, errStrategy := service.NewTradeStrategyFactory(input.Strategy, input.Params)
	if errStrategy != nil {
		return fmt.Errorf("invalid strategy: %s", err)
	}

	bot := entity.NewTradingBot(
		symbol,
		input.Quantity,
		strategy,
		input.IntervalSeconds,
		input.InitialCapital,
		input.TradeAmount,
		input.Currency,
		input.TradingFees,
		input.MinimumProfitThreshold,
		input.UseFixedQuantity,
	)

	errSave := uc.tradingBotRepository.Save(bot)
	if errSave != nil {
		return errSave
	}

	payload, errMarshal := json.Marshal(map[string]interface{}{
		"id":       bot.Id,
		"symbol":   bot.GetSymbol(),
		"quantity": bot.GetQuantity(),
		"strategy": bot.GetStrategy().GetName(),
	})
	if errMarshal != nil {
		return errMarshal
	}

	if err := uc.emitEvent("trading_bot.created", payload); err != nil {
		return err
	}

	return nil
}

func (uc *CreateTradingBotUseCase) emitEvent(topic string, payload []byte) error {
	fmt.Println("emitEvent ---- topic:", topic, "payload:", string(payload))
	event := queue.Message{
		RoutingKey: topic,
		Payload:    payload,
		Headers: map[string]string{
			"event_type": topic,
		},
	}
	if err := uc.messageBroker.Publish(uc.exchangeName, event); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}
	return nil
}
