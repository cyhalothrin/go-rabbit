package rabbit

import (
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ConsumerOption func(consumer *Consumer)

func ConsumerTag(tag string) ConsumerOption {
	return func(consumer *Consumer) {
		consumer.tag = tag
	}
}

func ConsumerLog(log logger) ConsumerOption {
	return func(consumer *Consumer) {
		consumer.log = log
	}
}

func GenerateConsumerTagPrefix(name string) ConsumerOption {
	return ConsumerTag(name + "-" + genRandomString(consumerTagRandomTailLength))
}

func ConsumerAutoAck() ConsumerOption {
	return func(consumer *Consumer) {
		consumer.autoAck = true
	}
}

func rpcConsumerOptions(responseExchange string, publisher *Publisher) ConsumerOption {
	return func(consumer *Consumer) {
		consumer.responseX = responseExchange
		consumer.publisher = publisher
	}
}

func ConsumerExclusive() ConsumerOption {
	return func(consumer *Consumer) {
		consumer.exclusive = true
	}
}

func ConsumerNoLocal() ConsumerOption {
	return func(consumer *Consumer) {
		consumer.noLocal = true
	}
}

func ConsumerNoWait() ConsumerOption {
	return func(consumer *Consumer) {
		consumer.noWait = true
	}
}

func ConsumerArgs(args amqp.Table) ConsumerOption {
	return func(consumer *Consumer) {
		consumer.args = args
	}
}

// ConsumerRPCResponsePublishTimeout set time for waiting response publish. Default 10 sec.
func ConsumerRPCResponsePublishTimeout(timeout time.Duration) ConsumerOption {
	return func(consumer *Consumer) {
		consumer.rpcResponsePublishTimeout = timeout
	}
}
