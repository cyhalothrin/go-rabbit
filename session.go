package rabbit

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Session wraps amqp091.Connection and watches for connection state changes and reconnects if necessary with
// declared exchanges, queues, bindings and consumers.
type Session struct {
	conn            *amqp.Connection
	errChan         chan error
	criticalErrChan chan error
	nextChan        chan struct{}
	// attempts is the number of times to attempt reconnecting to the server before giving up
	attempts  int
	serverNum int

	urls         []string
	conf         amqp.Config
	maxAttempts  int
	consumerConf *ConsumerConfig
	log          logger
	debug        bool

	declarations []Declaration
	consumers    *consumersGroup
	publisher    *Publisher
}

func NewSession(connectOptions ConnectionOptions, options ...SessionOption) (*Session, error) {
	urls := connectOptions.createURLs()
	if (len(urls)) == 0 {
		return nil, errors.New("no rabbitMQ servers specified")
	}

	errChan := make(chan error)
	criticalErrChan := make(chan error)
	s := &Session{
		urls:            urls,
		conf:            amqp.Config{},
		errChan:         errChan,
		criticalErrChan: criticalErrChan,
		nextChan:        make(chan struct{}),
		serverNum:       getRandomServerNum(len(urls)),
	}

	for _, opt := range options {
		opt(s)
	}

	if s.conf.Heartbeat == 0 {
		// default in rabbitmq/amqp091-go
		s.conf.Heartbeat = 10 * time.Second
	}

	if s.maxAttempts == 0 {
		s.maxAttempts = 100
	}

	if s.log == nil {
		s.log = &noopLogger{}
	} else {
		s.log = newLogWrapper(s.log, s.debug)
	}

	s.consumers = newConsumersGroup(errChan, criticalErrChan, s.log)

	return s, nil
}

// Start connects to the server and processes connection breaks until it receives a critical error, or until it
// reaches the maximum number of connection attempts.
func (s *Session) Start() error {
	defer s.Close()

	for {
		next, err := s.loop()
		if err != nil {
			return err
		}

		if next {
			continue
		}

		select {
		case err := <-s.criticalErrChan:
			return err
		case <-s.nextChan:
			s.Close()

			s.conn = nil
		}
	}
}

func (s *Session) Close() {
	if s.conn != nil {
		s.sendErr(s.conn.Close())
	}
}

// Publisher creates a new Publisher, method is not thread safe, don't use it in different goroutines, initialize it once at
// the start.
func (s *Session) Publisher(opts ...PublisherOption) *Publisher {
	if s.publisher == nil {
		s.publisher = newPublisher(opts...)
		s.publisher.log = s.log
	}

	return s.publisher
}

func (s *Session) AddConsumer(consumer *Consumer) {
	if consumer.log == nil {
		consumer.log = s.log
	} else {
		consumer.log = newLogWrapper(consumer.log, s.debug)
	}

	s.consumers.add(consumer)
}

// Declare declares Queue, Exchange and Binding. Declare first Exchange and Queue, then Binding,
// because it's just a list, and declare all in the specified order.
func (s *Session) Declare(declarations ...Declaration) {
	s.declarations = append(s.declarations, declarations...)
}

// loop handles connection and reinitializes all consumers, queues and exchanges.
func (s *Session) loop() (bool, error) {
	if s.conn != nil {
		return false, nil
	}

	conn, err := amqp.DialConfig(s.urls[s.serverNum], s.conf)
	if err != nil {
		s.log.Debug("connection failed ", err, " attempt:", s.attempts, s.urls[s.serverNum])

		s.serverNum = (s.serverNum + 1) % len(s.urls) // try next server
		s.sendErr(err)

		time.Sleep(backoff(s.attempts))

		s.attempts++
		if s.attempts == s.maxAttempts {
			return false, errors.New("maximum number of attempts to connect has been reached")
		}

		s.nextLoop()

		return true, nil
	}
	s.log.Debug("connected to ", s.urls[s.serverNum], " attempt:", s.attempts)

	s.conn = conn
	s.attempts = 0

	ch, err := conn.Channel()
	if err != nil {
		return false, fmt.Errorf("open channel failed: %w", err)
	}

	for _, declare := range s.declarations {
		err = declare(ch)
		if err != nil {
			return false, fmt.Errorf("declaration failed: %w", err)
		}
	}

	// RabbitMQ documentation has a recommendation to use different channels for publisher and consumer on the
	// same connection, but I don't know about many publishers or consumers on the same channel. maybe there
	// should be an option for that.
	// https://www.rabbitmq.com/channels.html#basics
	var isChannelUsed bool
	if !s.consumers.isEmpty() {
		err = s.configureConsumerChannel(ch)
		if err != nil {
			return false, err
		}

		s.consumers.initConsumers(ch, conn)
		isChannelUsed = true
	}

	if s.publisher != nil {
		if isChannelUsed {
			// create new channel for publisher
			ch, err = conn.Channel()
			if err != nil {
				return false, fmt.Errorf("open channel for publisher failed: %w", err)
			}
		}

		go s.publisher.serve(ch)
	}

	go s.watchConnection(conn)

	return false, nil
}

func (s *Session) configureConsumerChannel(ch *amqp.Channel) error {
	if s.consumerConf != nil {
		err := ch.Qos(s.consumerConf.PrefetchCount, s.consumerConf.PrefetchSize, s.consumerConf.PrefetchGlobal)
		if err != nil {
			return fmt.Errorf("consumer configuration: %w", err)
		}
	}

	return nil
}

func (s *Session) watchConnection(conn *amqp.Connection) {
	defer func() {
		s.nextLoop()

		if s.publisher != nil {
			s.publisher.stop()
		}
	}()

	reason, ok := <-conn.NotifyClose(make(chan *amqp.Error))
	if !ok {
		s.sendErr(errors.New("connection closed"))

		return
	}

	if reason != nil {
		s.sendErr(fmt.Errorf("close notify: %w", reason))
	}
}

func (s *Session) sendErr(err error) {
	if err == nil {
		return
	}

	select {
	case s.errChan <- err:
	default:
		s.log.Debug("unhandled error ", err)
	}
}

// Errors returns channel with all errors.
func (s *Session) Errors() <-chan error {
	return s.errChan
}

// nextLoop notifies that it's necessary to connect to the server again.
func (s *Session) nextLoop() {
	select {
	case s.nextChan <- struct{}{}:
	default:
	}
}

func getRandomServerNum(max int) int {
	src := rand.NewSource(time.Now().UnixNano())
	//nolint:gosec
	randomGen := rand.New(src)

	return randomGen.Intn(max)
}

func backoff(attempt int) time.Duration {
	timings := []int{1, 10, 20, 50, 100, 200, 300, 500, 1000}
	if len(timings) > attempt {
		return time.Duration(timings[attempt]) * time.Millisecond
	}

	return time.Second
}
