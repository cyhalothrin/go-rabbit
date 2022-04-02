package rabbit

import "time"

type RPCClientOption func(client *RPCClient)

// RPCClientRequestX sets exchange for calls
func RPCClientRequestX(name string) RPCClientOption {
	return func(client *RPCClient) {
		client.requestX = name
	}
}

// RPCClientResponseX sets exchange for responses
func RPCClientResponseX(name string) RPCClientOption {
	return func(client *RPCClient) {
		client.responseX = name
	}
}

// RPCClientCallTimeout see RPCClient.callTimeout
func RPCClientCallTimeout(timeout time.Duration) RPCClientOption {
	return func(client *RPCClient) {
		client.callTimeout = timeout
	}
}

// RPCClientLog sets own logger for client
func RPCClientLog(log logger) RPCClientOption {
	return func(client *RPCClient) {
		client.log = log
	}
}

// RPCClientReplyQueueName see RPCClient.replyQueueName
func RPCClientReplyQueueName(name string) RPCClientOption {
	return func(client *RPCClient) {
		client.replyQueueName = name
	}
}

// RPCClientReplyQueueNamePrefix see RPCClient.replyQueuePrefix
func RPCClientReplyQueueNamePrefix(name string) RPCClientOption {
	return func(client *RPCClient) {
		client.replyQueuePrefix = name
	}
}

type RPCServerOption func(client *RPCServer)

// RPCServerRequestX sets exchange for calls
func RPCServerRequestX(name string) RPCServerOption {
	return func(client *RPCServer) {
		client.requestX = name
	}
}

// RPCServerResponseX sets exchange for responses
func RPCServerResponseX(name string) RPCServerOption {
	return func(client *RPCServer) {
		client.responseX = name
	}
}
