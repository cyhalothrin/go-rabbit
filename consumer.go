package rabbit

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Handler interface {
	Handle(delivery Delivery) *Envelop
}

type HandlerFunc func(delivery Delivery) *Envelop

func (h HandlerFunc) Handle(delivery Delivery) *Envelop {
	return h(delivery)
}

const (
	consumerRandomTagLength     = 16
	consumerTagRandomTailLength = 10
)

type Consumer struct {
	handler                   Handler
	queue                     *Queue
	responseX                 string
	publisher                 *Publisher
	rpcResponsePublishTimeout time.Duration
	log                       logger
	// RabbitMQ's consumer options, you can use ConsumerOption for setting them
	tag       string
	autoAck   bool
	exclusive bool
	noLocal   bool
	noWait    bool
	args      amqp.Table

	stopc   chan struct{}
	errChan chan error
}

func NewConsumer(handler Handler, queue *Queue, opts ...ConsumerOption) *Consumer {
	c := &Consumer{
		handler: handler,
		queue:   queue,
		stopc:   make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.tag == "" {
		c.tag = genRandomString(consumerRandomTagLength)
	}

	if c.responseX == "" {
		c.responseX = defaultResX
	}

	if c.rpcResponsePublishTimeout == 0 {
		c.rpcResponsePublishTimeout = 10 * time.Second
	}

	return c
}

var errChanClosedUnexpectedly = errors.New("channel is closed unexpectedly")

func (c *Consumer) consume(ch *amqp.Channel) error {
	deliveriesc, err := ch.Consume(
		c.queue.Name,
		c.tag,
		c.autoAck,
		c.exclusive,
		c.noLocal,
		c.noWait,
		c.args,
	)
	if err != nil {
		return newErrCritical(fmt.Errorf("consume failed: %w", err))
	}

	defer func() {
		_ = ch.Cancel(c.tag, false)
	}()

	c.log.Debug("consumer started ", c.tag, c.queue.Name)

	for {
		select {
		case <-c.stopc:
			c.log.Info("consumer ", c.tag, " of ", c.queue.Name, " stopped")

			return nil
		case delivery, ok := <-deliveriesc:
			if !ok {
				return errChanClosedUnexpectedly
			}

			go func() {
				res := c.handler.Handle(delivery)
				c.log.Debug(
					"handled message exchange: ",
					delivery.Exchange,
					", queue: ",
					c.queue.Name,
					", routing_key: ",
					delivery.RoutingKey,
					", correlation_id: ",
					delivery.CorrelationId,
				)

				if res != nil {
					c.publishResponse(delivery, res)
				}
			}()
		}
	}
}

func (c *Consumer) publishResponse(delivery Delivery, res *Envelop) {
	if c.publisher == nil {
		c.log.Warn(
			"unexpected response from handler tag: ",
			c.tag,
			", queue: ",
			c.queue.Name,
			", routing_key: ",
			delivery.ReplyTo,
		)

		return
	}

	res.Exchange = c.responseX
	res.Payload.CorrelationId = delivery.CorrelationId
	res.Payload.ReplyTo = delivery.ReplyTo
	res.Payload.AppId = delivery.AppId
	res.Payload.Timestamp = time.Now()
	res.Key = delivery.ReplyTo

	ctx, cancel := context.WithTimeout(context.Background(), c.rpcResponsePublishTimeout)
	defer cancel()

	err := c.publisher.Publish(ctx, *res)
	if err != nil {
		c.sendErr(fmt.Errorf("publish RPC response: %w", err))
	}

	c.log.Debug(
		"call response published to: ",
		delivery.ReplyTo,
		", correlation_id: ",
		delivery.CorrelationId,
		", exchange: ",
		c.responseX,
		", error: ",
		err,
	)
}

func (c *Consumer) sendErr(err error) {
	select {
	case c.errChan <- err:
	default:
	}
}

func (c *Consumer) stop() {
	select {
	case c.stopc <- struct{}{}:
	default:
	}
}
