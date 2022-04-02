package rabbit

import (
	"errors"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type consumersGroup struct {
	sync.RWMutex
	consumers       map[string]*Consumer
	errChan         chan error
	criticalErrChan chan error
	log             logger
}

func newConsumersGroup(errChan, critErrChan chan error, log logger) *consumersGroup {
	return &consumersGroup{
		consumers:       make(map[string]*Consumer),
		errChan:         errChan,
		criticalErrChan: critErrChan,
		log:             log,
	}
}

func (c *consumersGroup) initConsumers(ch *amqp.Channel, conn *amqp.Connection) {
	var reInit sync.Once
	reInitFn := func() {
		c.log.Info("trying to reinitialize consumers")
		// if connection is closed, then the channel and consumers will be created when connection established
		if conn.IsClosed() {
			return
		}

		// stop all consumers
		c.stop()

		newChan, err := conn.Channel()
		if err != nil {
			c.criticalErr(fmt.Errorf("open new channel for consumer: %w", err))

			return
		}

		// trying to start all consumers on same connection with new channel
		c.initConsumers(newChan, conn)
	}

	c.iterate(func(consumer *Consumer) {
		go func() {
			err := consumer.consume(ch)
			if err == nil {
				return
			}
			if critErr, ok := asCritical(err); ok {
				c.criticalErr(critErr)

				return
			}

			c.sendErr(err)

			if errors.Is(err, errChanClosedUnexpectedly) {
				// maybe only channel is closed, because of unexpected reason
				reInit.Do(reInitFn)
			}
		}()
	})
}

func (c *consumersGroup) iterate(fn func(consumer *Consumer)) {
	c.RLock()
	defer c.RUnlock()

	for _, consumer := range c.consumers {
		fn(consumer)
	}
}

func (c *consumersGroup) stop() {
	c.iterate(func(consumer *Consumer) {
		consumer.stop()
	})
}

func (c *consumersGroup) isEmpty() bool {
	c.RLock()
	defer c.RUnlock()

	return len(c.consumers) == 0
}

func (c *consumersGroup) sendErr(err error) {
	select {
	case c.errChan <- err:
	default:
	}
}

func (c *consumersGroup) criticalErr(err error) {
	select {
	case c.criticalErrChan <- err:
	default:
	}
}

func (c *consumersGroup) add(consumer *Consumer) {
	c.Lock()
	defer c.Unlock()

	consumer.errChan = c.errChan
	c.consumers[consumer.tag] = consumer
}
