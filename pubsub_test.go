package rabbit_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cyhalothrin/go-rabbit"
)

//nolint:gochecknoglobals
var devRabbitConnection = rabbit.ConnectionOptions{
	IPs:         []string{"localhost"},
	Port:        5672,
	VirtualHost: "vhost",
	User:        "test",
	Password:    "test",
}

func TestPubSub(t *testing.T) {
	exc := &rabbit.Exchange{
		Name:       "amqp-lib-test-x",
		Kind:       "topic",
		AutoDelete: true,
		Durable:    false,
	}
	queue := &rabbit.Queue{
		Name:       "amqp-lib-test-queue",
		AutoDelete: true,
		Durable:    false,
	}
	bind := &rabbit.Binding{
		Queue:    queue,
		Exchange: exc,
		Key:      "hello",
	}

	helloMsg := "hello"
	consumeResult := make(chan string, 1)

	sess, err := rabbit.NewSession(
		devRabbitConnection,
		rabbit.SessionMaxAttempts(1),
		rabbit.SessionLog(&testLog{}, true),
	)
	if err != nil {
		t.Fatal(err)
	}

	sess.Declare(rabbit.DeclareExchange(exc), rabbit.DeclareQueue(queue), rabbit.DeclareBinding(bind))
	sess.AddConsumer(
		rabbit.NewConsumer(
			rabbit.HandlerFunc(func(d rabbit.Delivery) rabbit.Enveloper {
				consumeResult <- string(d.Body)

				return nil
			}),
			queue,
			rabbit.ConsumerAutoAck(),
		))

	pub := sess.Publisher()

	go func() {
		for err := range sess.Errors() {
			t.Error(err)
		}
	}()

	sessionErr := make(chan error, 1)
	go func() {
		exitErr := sess.Start()
		if exitErr != nil {
			sessionErr <- exitErr
		}
	}()

	// it needs for wait connection and declare queues
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = pub.Publish(ctx, &rabbit.Envelop{
		Payload: rabbit.Publishing{
			Body: []byte(helloMsg),
		},
		Exchange: exc.Name,
		Key:      "hello",
	})
	if err != nil {
		t.Fatal("publish:", err)
	}

	select {
	case res := <-consumeResult:
		if res != helloMsg {
			t.Fatalf("expected message %s, got: %s", helloMsg, res)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	case err := <-sessionErr:
		t.Fatal("session error:", err)
	}
}

type testLog struct{}

func (t *testLog) Debug(args ...interface{}) {
	fmt.Println(args...)
}

func (t *testLog) Info(args ...interface{}) {
	fmt.Println(args...)
}

func (t *testLog) Warn(args ...interface{}) {
	fmt.Println(args...)
}

func (t *testLog) Error(args ...interface{}) {
	fmt.Println(args...)
}
