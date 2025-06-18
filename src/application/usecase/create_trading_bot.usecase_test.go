package usecase

import (
	"errors"
	"github.com/adshao/go-binance/v2"
	"testing"

	"crypgo-machine/src/domain/entity"
)

type MockTradeBotRepository struct {
	SaveFunc func(*entity.TradeBot) error
}

func (m *MockTradeBotRepository) Save(bot *entity.TradeBot) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(bot)
	}
	return nil
}

func (m *MockTradeBotRepository) GetTradeByID(id string) (*entity.TradeBot, error) {
	return nil, nil
}

func (m *MockTradeBotRepository) Exists(id string) (bool, error) {
	return false, nil
}

func TestCreateTradingBotUseCase_Success(t *testing.T) {
	mockRepo := &MockTradeBotRepository{
		SaveFunc: func(bot *entity.TradeBot) error {
			if bot == nil {
				t.Error("bot cannot be nil")
			}
			if bot.Quantity() != 1.5 {
				t.Errorf("expected quantity 1.5, got %v", bot.Quantity())
			}
			if bot.Symbol() != "SOLBRL" {
				t.Errorf("expected symbol SOLBRL, got %s", bot.Symbol())
			}
			if bot.Strategy().Name() != "MovingAverage" {
				t.Errorf("expected strategy MovingAverage, got %s", bot.Strategy().Name())
			}
			return nil
		},
	}

	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{})

	input := Input{
		Symbol:   "SOLBRL",
		Quantity: 1.5,
		Strategy: "MovingAverage",
		Params:   MovingAverageParams{FastWindow: 7, SlowWindow: 21},
	}

	err := uc.Execute(input)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCreateTradingBotUseCase_InvalidSymbol(t *testing.T) {
	mockRepo := &MockTradeBotRepository{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{})

	input := Input{
		Symbol:   "INVALID",
		Quantity: 1,
		Strategy: "MovingAverage",
		Params:   MovingAverageParams{FastWindow: 7, SlowWindow: 21},
	}

	err := uc.Execute(input)
	if err == nil {
		t.Error("expected error for invalid symbol, got nil")
	}
}

func TestCreateTradingBotUseCase_RepoError(t *testing.T) {
	mockRepo := &MockTradeBotRepository{
		SaveFunc: func(bot *entity.TradeBot) error {
			return errors.New("db error")
		},
	}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{})

	input := Input{
		Symbol:   "SOLBRL",
		Quantity: 2.0,
		Strategy: "MovingAverage",
		Params:   MovingAverageParams{FastWindow: 10, SlowWindow: 20},
	}

	err := uc.Execute(input)
	if err == nil || err.Error() != "db error" {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestCreateTradingBotUseCase_InvalidQuantity(t *testing.T) {
	mockRepo := &MockTradeBotRepository{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{})

	input := Input{
		Symbol:   "SOLBRL",
		Quantity: 0, // Inválido
		Strategy: "MovingAverage",
		Params:   MovingAverageParams{FastWindow: 7, SlowWindow: 21},
	}

	err := uc.Execute(input)
	if err == nil || err.Error() == "" {
		t.Error("expected error for invalid quantity, got nil")
	}
}

func TestCreateTradingBotUseCase_UnknownStrategy(t *testing.T) {
	mockRepo := &MockTradeBotRepository{}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{})

	input := Input{
		Symbol:   "SOLBRL",
		Quantity: 1.0,
		Strategy: "NotARealStrategy", // Não existe
		Params:   MovingAverageParams{FastWindow: 7, SlowWindow: 21},
	}

	err := uc.Execute(input)
	if err == nil || err.Error() == "" {
		t.Error("expected error for unknown strategy, got nil")
	}
}

func TestCreateTradingBotUseCase_DuplicateBot(t *testing.T) {
	mockRepo := &MockTradeBotRepository{
		SaveFunc: func(bot *entity.TradeBot) error {
			return errors.New("bot already exists")
		},
	}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{})

	input := Input{
		Symbol:   "SOLBRL",
		Quantity: 1.0,
		Strategy: "MovingAverage",
		Params:   MovingAverageParams{FastWindow: 7, SlowWindow: 21},
	}

	err := uc.Execute(input)
	if err == nil || err.Error() != "bot already exists" {
		t.Errorf("expected 'bot already exists' error, got %v", err)
	}
}

func TestCreateTradingBotUseCase_MultipleBotsDifferentParams(t *testing.T) {
	callCount := 0
	mockRepo := &MockTradeBotRepository{
		SaveFunc: func(bot *entity.TradeBot) error {
			callCount++
			return nil
		},
	}
	uc := NewCreateTradingBotUseCase(mockRepo, binance.Client{})

	input1 := Input{
		Symbol:   "SOLBRL",
		Quantity: 1.0,
		Strategy: "MovingAverage",
		Params:   MovingAverageParams{FastWindow: 7, SlowWindow: 21},
	}
	input2 := Input{
		Symbol:   "SOLBRL",
		Quantity: 2.0,
		Strategy: "MovingAverage",
		Params:   MovingAverageParams{FastWindow: 9, SlowWindow: 30},
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
