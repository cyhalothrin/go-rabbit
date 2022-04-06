package rabbit

import (
	"net/url"
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SessionOption func(*Session)

//nolint:gocritic // amqp091 declares it by value
func SessionWithConfig(conf amqp.Config) SessionOption {
	return func(session *Session) {
		session.conf = conf
	}
}

// SessionMaxAttempts max number of connection attempts, default 100
func SessionMaxAttempts(max int) SessionOption {
	return func(session *Session) {
		session.maxAttempts = max
	}
}

// ConsumerConfig see channel.Qos()
type ConsumerConfig struct {
	PrefetchCount  int
	PrefetchSize   int
	PrefetchGlobal bool
}

// SessionConsumerConfig consumer config
func SessionConsumerConfig(conf *ConsumerConfig) SessionOption {
	return func(session *Session) {
		session.consumerConf = conf
	}
}

// SessionLog sets logger, it will be used by all consumers, rpc clients if not set for them. Default is noopLogger.
// debug enables logging internal events via logger.Debug()
func SessionLog(log logger, debug bool) SessionOption {
	return func(session *Session) {
		session.log = log
		session.debug = debug
	}
}

type ConnectionOptions struct {
	IPs         []string
	Port        int
	VirtualHost string
	User        string
	Password    string
}

func (c *ConnectionOptions) createURLs() (urls []string) {
	for _, ip := range c.IPs {
		link := url.URL{
			Scheme: "amqp",
			User:   url.UserPassword(c.User, c.Password),
			Host:   ip + ":" + strconv.Itoa(c.Port),
			Path:   c.VirtualHost,
		}
		urls = append(urls, link.String())
	}

	return urls
}
