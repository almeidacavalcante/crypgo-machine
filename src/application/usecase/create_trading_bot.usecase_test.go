package usecase

import (
	"crypgo-machine/src/domain/service"
	"crypgo-machine/src/infra/queue"
	"errors"
	"github.com/adshao/go-binance/v2"
	"testing"

	"crypgo-machine/src/domain/entity"
)

type MockTradeBotRepository struct {
	SaveFunc   func(bot *entity.TradingBot) error
	UpdateFunc func(bot *entity.TradingBot) error
}

func (m *MockTradeBotRepository) Update(bot *entity.TradingBot) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(bot)
	}
	return nil
}

func (m *MockTradeBotRepository) Save(bot *entity.TradingBot) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(bot)
	}
	return nil
}

func (m *MockTradeBotRepository) GetTradeByID(id string) (*entity.TradingBot, error) {
	return nil, nil
}

func (m *MockTradeBotRepository) Exists(id string) (bool, error) {
	return false, nil
}

func (m *MockTradeBotRepository) GetAllTradingBots() ([]*entity.TradingBot, error) {
	return nil, nil
}

func (m *MockTradeBotRepository) GetTradingBotsByStatus(status entity.Status) ([]*entity.TradingBot, error) {
	return nil, nil
}

// MockMessageBroker implements queue.MessageBroker for testing
type MockMessageBroker struct{}

func (m *MockMessageBroker) Publish(exchangeName string, message queue.Message) error {
	return nil
}

func (m *MockMessageBroker) Subscribe(exchangeName string, queueName string, routingKeys []string, handler func(msg queue.Message) error) error {
	return nil
}

func (m *MockMessageBroker) Close() error {
	return nil
}

func TestCreateTradingBotUseCase_Success(t *testing.T) {
	mockRepo := &MockTradeBotRepository{
		SaveFunc: func(bot *entity.TradingBot) error {
			if bot == nil {
				t.Error("bot cannot be nil")
			}
			if bot.GetQuantity() != 1.5 {
				t.Errorf("expected quantity 1.5, got %v", bot.GetQuantity())
			}
			if bot.GetSymbol().GetValue() != "SOLBRL" {
				t.Errorf("expected symbol SOLBRL, got %s", bot.GetSymbol())
			}
			if bot.GetStrategy().GetName() != "MovingAverage" {
				t.Errorf("expected strategy MovingAverage, got %s", bot.GetStrategy().GetName())
			}
			return nil
		},
	}

	mockMessageBroker := &MockMessageBroker{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{}, mockMessageBroker, "test-exchange")

	input := InputCreateTradingBot{
		Symbol:                   "SOLBRL",
		Quantity:                 1.5,
		Strategy:                 "MovingAverage",
		Params:                   service.MovingAverageParams{FastWindow: 7, SlowWindow: 21},
		IntervalSeconds:          3600,
		InitialCapital:           10000.0,
		TradeAmount:              4000.0,
		Currency:                 "BRL",
		TradingFees:              0.001,
		MinimumProfitThreshold:   5.0,
	}

	err := uc.Execute(input)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCreateTradingBotUseCase_InvalidSymbol(t *testing.T) {
	mockRepo := &MockTradeBotRepository{}
	mockMessageBroker := &MockMessageBroker{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{}, mockMessageBroker, "test-exchange")

	input := InputCreateTradingBot{
		Symbol:                   "INVALID",
		Quantity:                 1,
		Strategy:                 "MovingAverage",
		Params:                   service.MovingAverageParams{FastWindow: 7, SlowWindow: 21},
		IntervalSeconds:          3600,
		InitialCapital:           10000.0,
		TradeAmount:              4000.0,
		Currency:                 "BRL",
		TradingFees:              0.001,
		MinimumProfitThreshold:   5.0,
	}

	err := uc.Execute(input)
	if err == nil {
		t.Error("expected error for invalid symbol, got nil")
	}
}

func TestCreateTradingBotUseCase_RepoError(t *testing.T) {
	mockRepo := &MockTradeBotRepository{
		SaveFunc: func(bot *entity.TradingBot) error {
			return errors.New("db error")
		},
	}
	mockMessageBroker := &MockMessageBroker{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{}, mockMessageBroker, "test-exchange")

	input := InputCreateTradingBot{
		Symbol:                   "SOLBRL",
		Quantity:                 2.0,
		Strategy:                 "MovingAverage",
		Params:                   service.MovingAverageParams{FastWindow: 10, SlowWindow: 20},
		IntervalSeconds:          3600,
		InitialCapital:           10000.0,
		TradeAmount:              4000.0,
		Currency:                 "BRL",
		TradingFees:              0.001,
		MinimumProfitThreshold:   5.0,
	}

	err := uc.Execute(input)
	if err == nil || err.Error() != "db error" {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestCreateTradingBotUseCase_InvalidQuantity(t *testing.T) {
	mockRepo := &MockTradeBotRepository{}
	mockMessageBroker := &MockMessageBroker{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{}, mockMessageBroker, "test-exchange")

	input := InputCreateTradingBot{
		Symbol:                   "SOLBRL",
		Quantity:                 0, // Inválido
		Strategy:                 "MovingAverage",
		Params:                   service.MovingAverageParams{FastWindow: 7, SlowWindow: 21},
		IntervalSeconds:          3600,
		InitialCapital:           10000.0,
		TradeAmount:              4000.0,
		Currency:                 "BRL",
		TradingFees:              0.001,
		MinimumProfitThreshold:   5.0,
	}

	err := uc.Execute(input)
	if err == nil || err.Error() == "" {
		t.Error("expected error for invalid quantity, got nil")
	}
}

func TestCreateTradingBotUseCase_UnknownStrategy(t *testing.T) {
	mockRepo := &MockTradeBotRepository{}
	mockMessageBroker := &MockMessageBroker{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{}, mockMessageBroker, "test-exchange")

	input := InputCreateTradingBot{
		Symbol:                   "SOLBRL",
		Quantity:                 1.0,
		Strategy:                 "NotARealStrategy", // Não existe
		Params:                   service.MovingAverageParams{FastWindow: 7, SlowWindow: 21},
		IntervalSeconds:          3600,
		InitialCapital:           10000.0,
		TradeAmount:              4000.0,
		Currency:                 "BRL",
		TradingFees:              0.001,
		MinimumProfitThreshold:   5.0,
	}

	err := uc.Execute(input)
	if err == nil || err.Error() == "" {
		t.Error("expected error for unknown strategy, got nil")
	}
}

func TestCreateTradingBotUseCase_DuplicateBot(t *testing.T) {
	mockRepo := &MockTradeBotRepository{
		SaveFunc: func(bot *entity.TradingBot) error {
			return errors.New("bot already exists")
		},
	}
	mockMessageBroker := &MockMessageBroker{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{}, mockMessageBroker, "test-exchange")

	input := InputCreateTradingBot{
		Symbol:                   "SOLBRL",
		Quantity:                 1.0,
		Strategy:                 "MovingAverage",
		Params:                   service.MovingAverageParams{FastWindow: 7, SlowWindow: 21},
		IntervalSeconds:          3600,
		InitialCapital:           10000.0,
		TradeAmount:              4000.0,
		Currency:                 "BRL",
		TradingFees:              0.001,
		MinimumProfitThreshold:   5.0,
	}

	err := uc.Execute(input)
	if err == nil || err.Error() != "bot already exists" {
		t.Errorf("expected 'bot already exists' error, got %v", err)
	}
}

func TestCreateTradingBotUseCase_MultipleBotsDifferentParams(t *testing.T) {
	callCount := 0
	mockRepo := &MockTradeBotRepository{
		SaveFunc: func(bot *entity.TradingBot) error {
			callCount++
			return nil
		},
	}
	mockMessageBroker := &MockMessageBroker{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{}, mockMessageBroker, "test-exchange")

	input1 := InputCreateTradingBot{
		Symbol:                   "SOLBRL",
		Quantity:                 1.0,
		Strategy:                 "MovingAverage",
		Params:                   service.MovingAverageParams{FastWindow: 7, SlowWindow: 21},
		IntervalSeconds:          3600,
		InitialCapital:           10000.0,
		TradeAmount:              4000.0,
		Currency:                 "BRL",
		TradingFees:              0.001,
		MinimumProfitThreshold:   5.0,
	}
	input2 := InputCreateTradingBot{
		Symbol:                   "SOLBRL",
		Quantity:                 2.0,
		Strategy:                 "MovingAverage",
		Params:                   service.MovingAverageParams{FastWindow: 9, SlowWindow: 30},
		IntervalSeconds:          3600,
		InitialCapital:           15000.0,
		TradeAmount:              6000.0,
		Currency:                 "BRL",
		TradingFees:              0.001,
		MinimumProfitThreshold:   5.0,
	}

	err1 := uc.Execute(input1)
	err2 := uc.Execute(input2)
	if err1 != nil || err2 != nil {
		t.Errorf("expected no error, got err1: %v, err2: %v", err1, err2)
	}
	if callCount != 2 {
		t.Errorf("expected 2 saves, got %d", callCount)
	}
}

func TestCreateTradingBotUseCase_InvalidInitialCapital(t *testing.T) {
	mockRepo := &MockTradeBotRepository{}
	mockMessageBroker := &MockMessageBroker{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{}, mockMessageBroker, "test-exchange")

	input := InputCreateTradingBot{
		Symbol:                   "SOLBRL",
		Quantity:                 1.0,
		Strategy:                 "MovingAverage",
		Params:                   service.MovingAverageParams{FastWindow: 7, SlowWindow: 21},
		IntervalSeconds:          3600,
		InitialCapital:           0, // Invalid
		TradeAmount:              4000.0,
		Currency:                 "BRL",
		TradingFees:              0.001,
		MinimumProfitThreshold:   5.0,
	}

	err := uc.Execute(input)
	if err == nil || err.Error() != "invalid initial capital: must be greater than zero" {
		t.Errorf("expected initial capital error, got %v", err)
	}
}

func TestCreateTradingBotUseCase_InvalidTradeAmount(t *testing.T) {
	mockRepo := &MockTradeBotRepository{}
	mockMessageBroker := &MockMessageBroker{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{}, mockMessageBroker, "test-exchange")

	input := InputCreateTradingBot{
		Symbol:                   "SOLBRL",
		Quantity:                 1.0,
		Strategy:                 "MovingAverage",
		Params:                   service.MovingAverageParams{FastWindow: 7, SlowWindow: 21},
		IntervalSeconds:          3600,
		InitialCapital:           10000.0,
		TradeAmount:              -1000.0, // Invalid
		Currency:                 "BRL",
		TradingFees:              0.001,
		MinimumProfitThreshold:   5.0,
	}

	err := uc.Execute(input)
	if err == nil || err.Error() != "invalid trade amount: must be greater than zero" {
		t.Errorf("expected trade amount error, got %v", err)
	}
}

func TestCreateTradingBotUseCase_InvalidTradingFees(t *testing.T) {
	mockRepo := &MockTradeBotRepository{}
	mockMessageBroker := &MockMessageBroker{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{}, mockMessageBroker, "test-exchange")

	input := InputCreateTradingBot{
		Symbol:                   "SOLBRL",
		Quantity:                 1.0,
		Strategy:                 "MovingAverage",
		Params:                   service.MovingAverageParams{FastWindow: 7, SlowWindow: 21},
		IntervalSeconds:          3600,
		InitialCapital:           10000.0,
		TradeAmount:              4000.0,
		Currency:                 "BRL",
		TradingFees:              -0.001, // Invalid
		MinimumProfitThreshold:   5.0,
	}

	err := uc.Execute(input)
	if err == nil || err.Error() != "invalid trading fees: must be greater than or equal to zero" {
		t.Errorf("expected trading fees error, got %v", err)
	}
}

func TestCreateTradingBotUseCase_InvalidMinimumProfitThreshold(t *testing.T) {
	mockRepo := &MockTradeBotRepository{}
	mockMessageBroker := &MockMessageBroker{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{}, mockMessageBroker, "test-exchange")

	input := InputCreateTradingBot{
		Symbol:                   "SOLBRL",
		Quantity:                 1.0,
		Strategy:                 "MovingAverage",
		Params:                   service.MovingAverageParams{FastWindow: 7, SlowWindow: 21},
		IntervalSeconds:          3600,
		InitialCapital:           10000.0,
		TradeAmount:              4000.0,
		Currency:                 "BRL",
		TradingFees:              0.001,
		MinimumProfitThreshold:   -1.0, // Invalid
	}

	err := uc.Execute(input)
	if err == nil || err.Error() != "invalid minimum profit threshold: must be greater than or equal to zero" {
		t.Errorf("expected minimum profit threshold error, got %v", err)
	}
}
