package rabbit

import amqp "github.com/rabbitmq/amqp091-go"

type Queue struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp.Table
}

type Exchange struct {
	Name       string
	Kind       string
	Durable    bool
	AutoDelete bool
	Args       amqp.Table
}

type Binding struct {
	Queue    *Queue
	Exchange Exchange
	Key      string
	NoWait   bool
	Args     amqp.Table
}

// These aliases are added for convenience of not importing a rabbitmq/amqp091-go package
type (
	// Publishing alias of amqp091.Publishing
	Publishing = amqp.Publishing
	// Delivery alias of amqp091.Delivery
	Delivery = amqp.Delivery
)
