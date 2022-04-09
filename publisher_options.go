package rabbit

import "time"

type PublisherOption func(*Publisher)

// PublisherTimeout sets the timeout for publishing messages
func PublisherTimeout(timeout time.Duration) PublisherOption {
	return func(publisher *Publisher) {
		publisher.publishTimeout = timeout
	}
}
