package queue

import (
	"github.com/streadway/amqp"
)

type RabbitMQAdapter struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

func NewRabbitQMAdapter(url string) (*RabbitMQAdapter, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &RabbitMQAdapter{
		connection: conn,
		channel:    ch,
	}, nil
}

func (r *RabbitMQAdapter) Publish(exchangeName string, message Message) error {
	err := r.channel.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil)
	if err != nil {
		return err
	}

	amqpHeaders := make(amqp.Table)
	for k, v := range message.Headers {
		amqpHeaders[k] = v
	}

	return r.channel.Publish(
		exchangeName,
		message.RoutingKey,
		false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message.Payload,
			Headers:     amqpHeaders,
		},
	)
}

func (r *RabbitMQAdapter) Subscribe(
	exchangeName string,
	queueName string,
	routingKeys []string,
	handler func(message Message) error,
) error {
	err := r.channel.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil)
	if err != nil {
		return err
	}

	q, err := r.channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	for _, key := range routingKeys {
		if err := r.channel.QueueBind(q.Name, key, exchangeName, false, nil); err != nil {
			return err
		}
	}

	msgs, err := r.channel.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			handler(Message{
				RoutingKey: d.RoutingKey,
				Payload:    d.Body,
				Headers:    convertHeaders(d.Headers),
			})
		}
	}()

	return nil
}

func (r *RabbitMQAdapter) Close() error {
	if err := r.channel.Close(); err != nil {
		return err
	}
	return r.connection.Close()
}

func convertHeaders(hdr amqp.Table) map[string]string {
	out := make(map[string]string)
	for k, v := range hdr {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}
