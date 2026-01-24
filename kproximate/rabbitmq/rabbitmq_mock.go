package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ChannelMock struct {
	PendingMessages   int
	PublishedMessages int
}

func (c *ChannelMock) QueueDeclarePassive(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	return amqp.Queue{
		Name:     name,
		Messages: c.PendingMessages,
	}, nil
}

func (c *ChannelMock) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	c.PublishedMessages++
	return nil
}

type HttpClientMock struct {
	UnacknowledgedMessages int
}
