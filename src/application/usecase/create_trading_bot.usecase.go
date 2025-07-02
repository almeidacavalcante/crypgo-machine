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
	Symbol          string
	Quantity        float64
	Strategy        string
	Params          interface{}
	IntervalSeconds int
}

func (uc *CreateTradingBotUseCase) Execute(input InputCreateTradingBot) error {
	symbol, err := vo.NewSymbol(input.Symbol)
	if err != nil {
		return fmt.Errorf("invalid symbol: %s", err)
	}
	if input.Quantity <= 0 {
		return fmt.Errorf("invalid quantity: must be greater than zero")
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
