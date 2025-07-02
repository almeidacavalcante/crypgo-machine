package queue

type Message struct {
	RoutingKey string // ex: "trading_bot.created"
	Payload    []byte
	Headers    map[string]string
}

type MessageBroker interface {
	Publish(exchangeName string, message Message) error
	Subscribe(
		exchangeName string,
		queueName string,
		routingKeys []string,
		handler func(msg Message) error,
	) error
	Close() error
}
