package rabbit

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Declaration func(Declarer) error

// Declarer implemented by *amqp091.Channel
type Declarer interface {
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
}

func DeclareQueue(q *Queue) Declaration {
	return func(c Declarer) error {
		realQ, err := c.QueueDeclare(
			q.Name,
			q.Durable,
			q.AutoDelete,
			q.Exclusive,
			q.NoWait,
			q.Args,
		)
		q.Name = realQ.Name

		return fmt.Errorf("declare queue '%s': %w", q.Name, err)
	}
}

func DeclareExchange(e *Exchange) Declaration {
	return func(c Declarer) error {
		err := c.ExchangeDeclare(e.Name,
			e.Kind,
			e.Durable,
			e.AutoDelete,
			false,
			false,
			e.Args,
		)
		if err != nil {
			return fmt.Errorf("declare exchange '%s': %w", e.Name, err)
		}

		return nil
	}
}

func DeclareBinding(b *Binding) Declaration {
	return func(c Declarer) error {
		err := c.QueueBind(
			b.Queue.Name,
			b.Key,
			b.Exchange.Name,
			b.NoWait,
			b.Args,
		)
		if err != nil {
			return fmt.Errorf("bind queue %s to exhange %s: %w", b.Queue.Name, b.Exchange.Name, err)
		}

		return nil
	}
}
