package rabbit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	defaultReqX = "request"
	defaultResX = "response"
)

const (
	correlationIDLength        = 32
	replyQueueRandomTailLength = 32
)

// RPCClient with RabbitMQ you can create RPC with async call handling. It uses a queue with unique name for receiving
// responses. The name of the queue can be generated automatically, or you can specify it manually by
// RPCClientReplyQueueName, but it should be unique on server. Also, you can specify a prefix for the queue name.
type RPCClient struct {
	responseX  string
	requestX   string
	clientName string
	// callTimeout is the default timeout for call. It works together with ctx.Done() and not replaces it, so it makes
	// no sense to specify context.WithTimeout greater than this option. Use RPCClientCallTimeout to set it
	callTimeout      time.Duration
	replyQueueName   string
	replyQueuePrefix string
	session          *Session
	log              logger

	listener  *callResultListener
	publisher *Publisher
	stopc     chan struct{}
}

// NewRPCClient you can use clientName for identifying consumer, it will be added to consumer tag.
func NewRPCClient(clientName string, sess *Session, opts ...RPCClientOption) *RPCClient {
	stopc := make(chan struct{})
	r := &RPCClient{
		clientName: clientName,
		stopc:      stopc,
		session:    sess,
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.log == nil {
		r.log = sess.log
	} else {
		r.log = newLogWrapper(r.log, sess.debug)
	}

	r.listener = newCallResultListener(stopc, r.log)
	r.init()

	return r
}

func (r *RPCClient) init() {
	if r.responseX == "" {
		r.responseX = defaultResX
	}

	if r.requestX == "" {
		r.requestX = defaultReqX
	}

	if r.replyQueueName == "" {
		prefix := r.replyQueuePrefix
		if prefix == "" {
			prefix = r.responseX
		}

		r.replyQueueName = fmt.Sprintf(
			"%s.%s.%s",
			prefix,
			r.clientName,
			genRandomString(replyQueueRandomTailLength),
		)
		r.log.Debug("generated reply queue ", r.replyQueueName)
	}

	if r.callTimeout == 0 {
		r.callTimeout = 10 * time.Second
	}

	// this queue will be used for receiving responses
	replyQueue := &Queue{
		Name:       r.replyQueueName,
		Durable:    false,
		AutoDelete: true,
		Exclusive:  false,
		NoWait:     false,
		Args:       nil,
	}

	r.session.Declare(
		DeclareQueue(replyQueue),
		DeclareBinding(Binding{
			Queue:    replyQueue,
			Exchange: Exchange{Name: r.responseX},
			Key:      r.replyQueueName,
			Args:     nil,
		}),
	)

	repeater := NewConsumer(
		r.listener,
		replyQueue,
		GenerateConsumerTagPrefix("listener."+r.clientName),
		ConsumerAutoAck(),
	)
	r.session.AddConsumer(repeater)

	r.publisher = r.session.Publisher()
}

// Call calls the remote handler by routing key, ctx is used for canceling or timeout of call, see RPCClient.callTimeout
func (r *RPCClient) Call(ctx context.Context, key string, msg Enveloper) (Delivery, error) {
	envelop, err := msg.Envelop()
	if err != nil {
		return Delivery{}, err
	}

	replyCh := make(chan Delivery)
	correlationID := genRandomString(correlationIDLength)

	r.log.Debug("call key: ", key, ", correlation_id: ", correlationID)

	r.listener.put(correlationID, replyCh)
	defer r.listener.closeChan(correlationID)

	envelop.Exchange = r.requestX
	envelop.Key = key
	envelop.Payload.CorrelationId = correlationID
	envelop.Payload.ReplyTo = r.replyQueueName
	envelop.Mandatory = false
	envelop.Immediate = false

	if err := r.publisher.Publish(ctx, envelop); err != nil {
		return Delivery{}, fmt.Errorf("publish RPC request: %w", err)
	}

	r.log.Debug("call published correlation_id: ", correlationID)

	select {
	case reply := <-replyCh:
		r.log.Debug("received response correlation_id: ", correlationID)

		return reply, nil
	case <-time.After(r.callTimeout):
		return Delivery{}, fmt.Errorf("RPC response %s: %w", correlationID, ErrTimeout)
	case <-ctx.Done():
		return Delivery{}, fmt.Errorf("RPC response %s: %w", correlationID, ctx.Err())
	case <-r.stopc:
		return Delivery{}, errors.New("stopped")
	}
}

// ReplyQueueName if you need to know the name of reply queue for some reasons
func (r *RPCClient) ReplyQueueName() string {
	return r.replyQueueName
}

func (r *RPCClient) Close() {
	close(r.stopc)
	// QueueUnbind?
	r.listener.close()
}

// callResultListener sends incoming messages to channels waiting for response, by CorrelationId of message
type callResultListener struct {
	sync.RWMutex
	listeners map[string]chan Delivery
	stopc     chan struct{}
	log       logger
}

func newCallResultListener(stopc chan struct{}, log logger) *callResultListener {
	return &callResultListener{
		listeners: make(map[string]chan Delivery),
		stopc:     stopc,
		log:       log,
	}
}

func (c *callResultListener) Handle(delivery Delivery) *Envelop {
	c.RLock()
	defer c.RUnlock()

	listener, ok := c.listeners[delivery.CorrelationId]
	if ok {
		select {
		case listener <- delivery:
			c.log.Debug("sent response to caller correlation_id: ", delivery.CorrelationId)
		case <-c.stopc:
			c.log.Info("RPC reply queue handler was stopped")
		default:
			c.log.Warn("RPC response not delivered correlation_id: ", delivery.CorrelationId, ", routing_key: ", delivery.ReplyTo)
		}
	} else {
		c.log.Warn("no listener for correlation_id: ", delivery.CorrelationId)
	}

	return nil
}

func (c *callResultListener) closeChan(id string) {
	c.Lock()
	defer c.Unlock()

	if ch, ok := c.listeners[id]; ok {
		close(ch)
		delete(c.listeners, id)
	}
}

func (c *callResultListener) put(id string, replyCh chan Delivery) {
	c.Lock()
	c.listeners[id] = replyCh
	c.Unlock()
}

func (c *callResultListener) close() {
	c.Lock()
	for k, ch := range c.listeners {
		close(ch)
		delete(c.listeners, k)
	}
	c.Unlock()
}
