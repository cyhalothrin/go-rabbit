package rabbit

import (
	"context"
	"log"
	"testing"
	"time"
)

//nolint:gochecknoglobals
var devRabbitConnection = ConnectionOptions{
	IPs:         []string{"10.252.16.116"},
	Port:        5672,
	VirtualHost: "",
	User:        "",
	Password:    "",
}

func TestPubSub(t *testing.T) {
	exc := &Exchange{
		Name:       "amqp-lib-test-x",
		Kind:       "topic",
		AutoDelete: true,
		Durable:    false,
	}
	queue := &Queue{
		Name:       "amqp-lib-test-queue",
		AutoDelete: true,
		Durable:    false,
	}
	bind := &Binding{
		Queue:    queue,
		Exchange: exc,
		Key:      "hello",
	}

	helloMsg := "hello-" + genRandomString(16)
	consumeResult := make(chan string, 1)

	sess, err := NewSession(devRabbitConnection, SessionMaxAttempts(1), SessionLog(&testLog{}, true))
	if err != nil {
		t.Fatal(err)
	}

	sess.Declare(DeclareExchange(exc), DeclareQueue(queue), DeclareBinding(bind))
	sess.AddConsumer(
		NewConsumer(
			HandlerFunc(func(d Delivery) *Envelop {
				consumeResult <- string(d.Body)

				return nil
			}),
			queue,
			ConsumerAutoAck(),
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

	err = pub.Publish(ctx, &Envelop{
		Payload: Publishing{
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
	log.Println(args...)
}

func (t *testLog) Info(args ...interface{}) {
	log.Println(args...)
}

func (t *testLog) Warn(args ...interface{}) {
	log.Println(args...)
}

func (t *testLog) Error(args ...interface{}) {
	log.Println(args...)
}
