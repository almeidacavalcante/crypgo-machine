package usecase

import (
	"crypgo-machine/src/domain/entity"
	"crypgo-machine/src/infra/queue"
	"crypgo-machine/src/infra/repository"
	"encoding/json"
	"errors"
	"sync"
	"testing"
)

// recordingLifecycleBroker records published messages and can be configured to fail.
type recordingLifecycleBroker struct {
	mu         sync.Mutex
	messages   []queue.Message
	exchanges  []string
	publishErr error
}

func (b *recordingLifecycleBroker) Publish(exchangeName string, message queue.Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages = append(b.messages, message)
	b.exchanges = append(b.exchanges, exchangeName)
	return b.publishErr
}

func (b *recordingLifecycleBroker) Subscribe(exchangeName string, queueName string, routingKeys []string, handler func(msg queue.Message) error) error {
	return nil
}

func (b *recordingLifecycleBroker) Close() error {
	return nil
}

// messagesWithKey returns the published messages with the given routing key and the exchanges they went to.
func (b *recordingLifecycleBroker) messagesWithKey(routingKey string) ([]queue.Message, []string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var msgs []queue.Message
	var exchanges []string
	for i, m := range b.messages {
		if m.RoutingKey == routingKey {
			msgs = append(msgs, m)
			exchanges = append(exchanges, b.exchanges[i])
		}
	}
	return msgs, exchanges
}

// assertLifecyclePayload checks the fields the email and Telegram consumers read.
func assertLifecyclePayload(t *testing.T, msg queue.Message, bot *entity.TradingBot) {
	t.Helper()
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	if payload["id"] != bot.Id.GetValue() {
		t.Errorf("expected payload id %q, got %v", bot.Id.GetValue(), payload["id"])
	}
	if payload["symbol"] != bot.GetSymbol().GetValue() {
		t.Errorf("expected payload symbol %q, got %v", bot.GetSymbol().GetValue(), payload["symbol"])
	}
	if msg.Headers["event_type"] != msg.RoutingKey {
		t.Errorf("expected event_type header %q, got %q", msg.RoutingKey, msg.Headers["event_type"])
	}
}

func saveRunningTestBot(t *testing.T, repo *repository.TradeBotRepositoryInMemory) *entity.TradingBot {
	t.Helper()
	bot := createTestTradingBot()
	if err := bot.Start(); err != nil {
		t.Fatalf("failed to start bot: %v", err)
	}
	if err := repo.Save(bot); err != nil {
		t.Fatalf("failed to save bot: %v", err)
	}
	return bot
}

func TestStopTradingBotUseCase_PublishesStoppedEvent(t *testing.T) {
	repo := repository.NewTradeBotRepositoryInMemory()
	broker := &recordingLifecycleBroker{}
	uc := NewStopTradingBotUseCaseWithMessaging(repo, broker, "test-exchange")

	bot := saveRunningTestBot(t, repo)

	if err := uc.Execute(InputStopTradingBot{BotId: bot.Id.GetValue()}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stored, _ := repo.GetTradeByID(bot.Id.GetValue())
	if stored.GetStatus() != entity.StatusStopped {
		t.Errorf("expected status %s, got %s", entity.StatusStopped, stored.GetStatus())
	}

	msgs, exchanges := broker.messagesWithKey("trading_bot.stopped")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 trading_bot.stopped event, got %d", len(msgs))
	}
	if exchanges[0] != "test-exchange" {
		t.Errorf("expected exchange test-exchange, got %s", exchanges[0])
	}
	assertLifecyclePayload(t, msgs[0], bot)
}

func TestStopTradingBotUseCase_PublishFailureDoesNotFailStop(t *testing.T) {
	repo := repository.NewTradeBotRepositoryInMemory()
	broker := &recordingLifecycleBroker{publishErr: errors.New("broker down")}
	uc := NewStopTradingBotUseCaseWithMessaging(repo, broker, "test-exchange")

	bot := saveRunningTestBot(t, repo)

	if err := uc.Execute(InputStopTradingBot{BotId: bot.Id.GetValue()}); err != nil {
		t.Fatalf("expected stop to succeed despite publish failure, got %v", err)
	}

	stored, _ := repo.GetTradeByID(bot.Id.GetValue())
	if stored.GetStatus() != entity.StatusStopped {
		t.Errorf("expected status %s, got %s", entity.StatusStopped, stored.GetStatus())
	}
}

func TestStopTradingBotUseCase_WithoutBroker(t *testing.T) {
	repo := repository.NewTradeBotRepositoryInMemory()
	uc := NewStopTradingBotUseCase(repo)

	bot := saveRunningTestBot(t, repo)

	if err := uc.Execute(InputStopTradingBot{BotId: bot.Id.GetValue()}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stored, _ := repo.GetTradeByID(bot.Id.GetValue())
	if stored.GetStatus() != entity.StatusStopped {
		t.Errorf("expected status %s, got %s", entity.StatusStopped, stored.GetStatus())
	}
}

func TestStopTradingBotUseCase_AlreadyStoppedDoesNotPublish(t *testing.T) {
	repo := repository.NewTradeBotRepositoryInMemory()
	broker := &recordingLifecycleBroker{}
	uc := NewStopTradingBotUseCaseWithMessaging(repo, broker, "test-exchange")

	bot := createTestTradingBot()
	if err := repo.Save(bot); err != nil {
		t.Fatalf("failed to save bot: %v", err)
	}

	if err := uc.Execute(InputStopTradingBot{BotId: bot.Id.GetValue()}); err == nil {
		t.Fatal("expected error for already stopped bot, got nil")
	}

	if msgs, _ := broker.messagesWithKey("trading_bot.stopped"); len(msgs) != 0 {
		t.Errorf("expected no trading_bot.stopped event, got %d", len(msgs))
	}
}
