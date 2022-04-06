package rabbit

type RPCServer struct {
	session    *Session
	requestX   string
	responseX  string
	clientName string
}

func NewRPCServer(clientName string, session *Session, opts ...RPCServerOption) *RPCServer {
	srv := &RPCServer{
		session:    session,
		clientName: clientName,
	}

	for _, opt := range opts {
		opt(srv)
	}

	if srv.responseX == "" {
		srv.responseX = defaultResX
	}

	if srv.requestX == "" {
		srv.requestX = defaultReqX
	}

	return srv
}

// Handle sets handler for specified endpoint. Handler must return non-empty response
func (r *RPCServer) Handle(endpoint string, handler Handler) {
	queue := &Queue{
		Name:       endpoint,
		Durable:    true,
		AutoDelete: true,
		Exclusive:  false,
		NoWait:     true,
	}
	bind := &Binding{
		Queue:    queue,
		Exchange: &Exchange{Name: r.requestX},
		Key:      endpoint,
	}

	r.session.Declare(DeclareQueue(queue), DeclareBinding(bind))
	r.session.AddConsumer(
		NewConsumer(
			handler,
			queue,
			ConsumerAutoAck(),
			ConsumerNoWait(),
			rpcConsumerOptions(r.responseX, r.session.Publisher()),
			GenerateConsumerTagPrefix("handler."+r.clientName+"."+endpoint),
		),
	)
}
