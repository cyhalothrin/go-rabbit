package rabbit_test

import (
	"context"
	"testing"
	"time"

	"github.com/cyhalothrin/go-rabbit"
)

const exchange = "amq.topic"

func TestRPC(t *testing.T) {
	sess, err := rabbit.NewSession(
		devRabbitConnection,
		rabbit.SessionMaxAttempts(1),
		rabbit.SessionLog(&testLog{}, true),
	)
	if err != nil {
		t.Fatal(err)
	}

	clientMessage := "hello-server"
	serverReply := "hello-client"
	endpoint := "ampq-lib-test-endpoint"

	rpcSrv := rabbit.NewRPCServer(
		"hello-server",
		sess,
		rabbit.RPCServerRequestX(exchange),
		rabbit.RPCServerResponseX(exchange),
	)
	rpcSrv.Handle(endpoint, rabbit.HandlerFunc(func(delivery rabbit.Delivery) rabbit.Enveloper {
		if clientMessage != string(delivery.Body) {
			t.Fatalf("expected client message %s, got %s", clientMessage, string(delivery.Body))

			return nil
		}

		return &rabbit.Envelop{
			Payload: rabbit.Publishing{Body: []byte(serverReply)},
		}
	}))

	rpcClient := rabbit.NewRPCClient(
		"hello-client",
		sess,
		rabbit.RPCClientRequestX(exchange),
		rabbit.RPCClientResponseX(exchange),
		rabbit.RPCClientCallTimeout(30*time.Second),
		rabbit.RPCClientReplyQueueName("hello-client-reply-queue"),
	)

	go func() {
		for err := range sess.Errors() {
			t.Error(err)
		}
	}()

	sessionErr := make(chan error, 1)
	go func() {
		err := sess.Start()
		if err != nil {
			sessionErr <- err
		}
	}()

	replyErrc := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := rpcClient.Call(ctx, endpoint, &rabbit.Envelop{
			Payload: rabbit.Publishing{Body: []byte(clientMessage)},
		})
		if err != nil {
			replyErrc <- err

			return
		}

		if string(res.Body) != serverReply {
			t.Errorf("expected server response %s, got %s", res.Body, serverReply)

			return
		}

		replyErrc <- nil
	}()

	select {
	case <-time.After(10 * time.Second):
		t.Fatal("timeout")
	case err := <-sessionErr:
		if err != nil {
			t.Fatal("session error:", err)
		}
	case err := <-replyErrc:
		if err != nil {
			t.Fatal("response:", err)
		}
	}
}
