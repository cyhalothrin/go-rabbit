package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cyhalothrin/go-rabbit"
)

func main() {
	exc := &rabbit.Exchange{
		Name:       "myExc",
		Kind:       "fanout",
		AutoDelete: true,
	}
	queue := &rabbit.Queue{
		Name: "test-queue",
	}
	bind := &rabbit.Binding{
		Queue:    queue,
		Exchange: exc,
		Key:      "hello",
	}

	connOpt := rabbit.ConnectionOptions{
		IPs:         []string{"localhost"},
		Port:        5672,
		VirtualHost: "dev-onetech",
		User:        "guest",
		Password:    "guest",
	}

	donec := make(chan struct{})

	sess, _ := rabbit.NewSession(connOpt, rabbit.SessionLog(&testLog{}, true))
	sess.Declare(rabbit.DeclareExchange(exc), rabbit.DeclareQueue(queue), rabbit.DeclareBinding(bind))
	sess.AddConsumer(
		rabbit.NewConsumer(
			rabbit.HandlerFunc(func(d rabbit.Delivery) rabbit.Enveloper {
				// possible panic if, for some reason, rabbit has 2 messages
				defer close(donec)

				fmt.Println("consume", string(d.Body))

				_ = d.Ack(true)

				return nil
			}),
			queue,
		))

	pub := sess.Publisher()

	go func() {
		for err := range sess.Errors() {
			log.Println(err)
		}
	}()

	go func() {
		err := sess.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()

	//nolint:govet
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	err := pub.Publish(ctx, &rabbit.Envelop{
		Payload: rabbit.Publishing{
			Body: []byte("hello"),
		},
		Exchange: exc.Name,
		Key:      "hello",
	})
	log.Println(err)

	select {
	case <-time.After(20 * time.Second):
		log.Println("timeout")
	case <-donec:
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
