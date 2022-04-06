package rabbit

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	stopc          chan struct{}
	pubc           chan publishCarrier
	publishTimeout time.Duration
	log            logger
}

func newPublisher(opts ...PublisherOption) *Publisher {
	p := &Publisher{
		pubc:  make(chan publishCarrier),
		stopc: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.publishTimeout == 0 {
		p.publishTimeout = 10 * time.Second
	}

	return p
}

// serve listens to the pubc channel and publishes messages to the given amqp.Channel
func (p *Publisher) serve(ch *amqp.Channel) {
	p.log.Debug("publisher started")

	for {
		select {
		case <-p.stopc:
			return
		case pub := <-p.pubc:
			err := p.publish(ch, pub.payload)
			pub.replayc <- err // channel is buffered
			p.log.Debug(
				"sent result back to publisher key: ",
				pub.payload.Key,
				", correlation_id: ",
				pub.payload.Payload.CorrelationId,
				", error: ",
				err,
			)
		}
	}
}

// Publish you can pass context for timeout or cancellation
func (p *Publisher) Publish(ctx context.Context, msg Enveloper) error {
	envelop, err := msg.Envelop()
	if err != nil {
		return err
	}

	pub := newPublishCarrier(envelop)
	timeoutc := time.After(p.publishTimeout)

	select {
	case <-ctx.Done():
		return fmt.Errorf("send carrier %s: %w", pub.payload.Payload.CorrelationId, ctx.Err())
	case <-timeoutc:
		return fmt.Errorf("send carrier %s: %w", pub.payload.Payload.CorrelationId, ErrTimeout)
	case p.pubc <- pub:
		p.log.Debug("sent message to GO publisher channel ", pub.payload.Payload.CorrelationId)

		select {
		case res := <-pub.replayc:
			p.log.Debug("received publish result: ", res)

			return res
		case <-ctx.Done():
			return fmt.Errorf("waiting response from carrier %s: %w", pub.payload.Payload.CorrelationId, ctx.Err())
		case <-timeoutc:
			return fmt.Errorf("waiting response from carrier %s: %w", pub.payload.Payload.CorrelationId, ErrTimeout)
		}
	}
}

func (p *Publisher) publish(ch *amqp.Channel, envelop *Envelop) error {
	return ch.Publish(
		envelop.Exchange,
		envelop.Key,
		envelop.Mandatory,
		envelop.Immediate,
		envelop.Payload,
	)
}

func (p *Publisher) stop() {
	select {
	case p.stopc <- struct{}{}:
	default:
	}
}

type publishCarrier struct {
	// replayc channel with results of publishing
	replayc chan error
	payload *Envelop
}

func newPublishCarrier(envelop *Envelop) publishCarrier {
	return publishCarrier{
		payload: envelop,
		// replayc is buffered channel to prevent deadlock
		replayc: make(chan error, 1),
	}
}
