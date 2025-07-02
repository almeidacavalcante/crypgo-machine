package notification

import (
	"encoding/json"
	"fmt"
	"log"

	"crypgo-machine/src/infra/queue"
)

type EmailNotificationConsumer struct {
	broker       queue.MessageBroker
	exchangeName string
	queueName    string
}

func NewEmailNotificationConsumer(
	broker queue.MessageBroker,
	exchangeName string,
	queueName string,
) *EmailNotificationConsumer {
	return &EmailNotificationConsumer{
		broker:       broker,
		exchangeName: exchangeName,
		queueName:    queueName,
	}
}

func (e *EmailNotificationConsumer) Start() error {
	routingKeys := []string{
		"trading_bot.created",
		"trading_bot.started",
		"trading_bot.stopped",
	}

	return e.broker.Subscribe(e.exchangeName, e.queueName, routingKeys, e.handleMessage)
}

func (e *EmailNotificationConsumer) handleMessage(msg queue.Message) error {
	var payload map[string]interface{}
	_ = json.Unmarshal(msg.Payload, &payload)

	to := "user@example.com"
	subject := ""
	body := ""

	switch msg.RoutingKey {
	case "trading_bot.created":
		subject = "Seu trading bot foi criado"
		body = fmt.Sprintf("O bot %v foi criado com sucesso.", payload["id"])
	case "trading_bot.started":
		subject = "Seu trading bot foi iniciado"
		body = fmt.Sprintf("O bot %v está agora rodando.", payload["id"])
	case "trading_bot.stopped":
		subject = "Seu trading bot foi pausado"
		body = fmt.Sprintf("O bot %v foi interrompido.", payload["id"])
	default:
		log.Println("Evento ignorado:", msg.RoutingKey)
		return nil
	}

	log.Println("📬 Simulando envio de email:")
	log.Println("To:", to)
	log.Println("Subject:", subject)
	log.Println("Body:", body)
	log.Println("---")

	return nil
}
