package rabbit

import (
	"context"
	"testing"
	"time"
)

func TestRPC(t *testing.T) {
	sess, err := NewSession(devRabbitConnection, SessionMaxAttempts(1))
	if err != nil {
		t.Fatal(err)
	}

	clientMessage := "hello-server-" + genRandomString(10)
	serverReply := "hello-client-" + genRandomString(10)

	endpoint := "ampq-lib-test-endpoint"

	rpcSrv := NewRPCServer(
		"hello-server",
		sess,
		RPCServerRequestX("X:routing.topic"),
		RPCServerResponseX("X:gateway.out.fanout"),
	)
	rpcSrv.Handle(endpoint, HandlerFunc(func(delivery Delivery) *Envelop {
		if clientMessage != string(delivery.Body) {
			t.Fatalf("expected client message %s, got %s", clientMessage, string(delivery.Body))

			return nil
		}

		return &Envelop{
			Payload: Publishing{Body: []byte(serverReply)},
		}
	}))

	rpcClient := NewRPCClient(
		"hello-client",
		sess,
		RPCClientRequestX("X:gateway.in.fanout"),
		RPCClientResponseX("X:routing.topic"),
		RPCClientCallTimeout(30*time.Second),
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

		res, err := rpcClient.Call(ctx, endpoint, &Envelop{
			Payload: Publishing{Body: []byte(clientMessage)},
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
		t.Fatal("session error:", err)
	case err := <-replyErrc:
		if err != nil {
			t.Fatal("response:", err)
		}
	}
}
