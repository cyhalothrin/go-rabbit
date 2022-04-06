package main

import (
	"context"
	"log"
	"time"

	"github.com/cyhalothrin/go-rabbit"
)

func main() {
	connOpt := rabbit.ConnectionOptions{
		IPs:         []string{"localhost"},
		Port:        5672,
		VirtualHost: "dev-onetech",
		User:        "guest",
		Password:    "guest",
	}
	sess, _ := rabbit.NewSession(connOpt)

	rpcSrv := rabbit.NewRPCServer(
		"hello-server",
		sess,
		rabbit.RPCServerRequestX("X:routing.topic"),
		rabbit.RPCServerResponseX("X:gateway.out.fanout"),
	)
	rpcSrv.Handle("hello", rabbit.HandlerFunc(func(delivery rabbit.Delivery) *rabbit.Envelop {
		log.Println("request:", string(delivery.Body))

		return &rabbit.Envelop{
			Payload: rabbit.Publishing{
				Body: []byte("hello client"),
			},
		}
	}))

	rpcClient := rabbit.NewRPCClient(
		"hello-client",
		sess,
		rabbit.RPCClientRequestX("X:routing.topic"),
		rabbit.RPCClientResponseX("X:gateway.out.fanout"),
	)

	go func() {
		if err := sess.Start(); err != nil {
			log.Fatal("session:", err)
		}
	}()
	go func() {
		for err := range sess.Errors() {
			log.Println("session:", err)
		}
	}()

	donec := make(chan struct{})
	go func() {
		defer func() {
			donec <- struct{}{}
		}()

		for i := 0; i < 100; i++ {
			//nolint:govet
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
			res, err := rpcClient.Call(ctx, "hello", &rabbit.Envelop{
				Payload: rabbit.Publishing{
					Body: []byte("hello server"),
				},
			})

			if err != nil {
				log.Println("call error:", err)
			} else {
				log.Println("response:", string(res.Body))
			}

			time.Sleep(time.Second)
		}
	}()

	<-donec
}
