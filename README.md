# go-rabbit

Golang AMQP client with reconnection logic. Uses [rabbitmq/amqp091-go](https://github.com/rabbitmq/amqp091-go) under the hood.

# Installation

```bash
go get github.com/cyhalothrin/go-rabbit@latest
```

# Quick Start

## Consumer

First, we need to declare queue and binding, define consumers and publishers, then call `session.Start()` to connect 
to broker and watch session lifecycle. Channels will be recreated on reconnection with defined consumers and publishers.

```go
opt := rabbit.ConnectionOptions{
    IPs:         []string{"localhost"},
    Port:        5672,
    VirtualHost: "default",
    User:        "guest",
    Password:    "guest",
}

// connection is not established at this moment
session, err := rabbit.NewSession(opt)
if err != nil {
    panic(err)
}

ex := Exchange{ Name: "amq.topic" }
queue := &Queue{ Name: "amqp-lib-test-queue" }
bind := Binding{
    Queue:    queue,
    Exchange: exc,
    Key:      "hello",
}

sess.Declare(rabbit.DeclareExchange(ex), rabbit.DeclareQueue(queue), rabbit.DeclareBinding(bind))

sess.AddConsumer(
    NewConsumer(
        HandlerFunc(func(d Delivery) *Envelop {
            fmt.Println("Received:", string(d.Body))
			
            return nil
    }),
    queue,
    ConsumerAutoAck(),
))

// connect to the server and watch connection status
err := sess.Start()
if err != nil {
    panic(err)
}
```

`session.Start()` will try to connect to the broker until all attempts fail. In second case, returns error.
You need to handle this error, but it usually means that the application can no longer work with the broker. 
The number of attempts can be defined by `SessionMaxAttempts` option, default 100.

## Publisher

Publisher shares the same connection as consumer, but will use a different channel. 
See RabbitMQ documentation for more [details](https://www.rabbitmq.com/channels.html#basics).

```go
// create session as above, you can reuse the same session
pub := session.Publisher()

err := pub.Publish(ctx, rabbit.Envelop{
        Payload: rabbit.Publishing{
        Body: []byte("hello"),
    },
    Exchange: exc.Name,
    Key:      "hello",
})
if err != nil {
	panic(err)
}
```

Call session.Start() after all consumers and publishers are declared, but only call it once.

## RPC
See the [examples](example/rpc) directory for more details.

## Tests

Run rabbit:

```
make rabbit
```

and tests:

```
make test
```