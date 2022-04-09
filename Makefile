lint:
	golangci-lint run

test:
	go test

rabbit:
	docker run -d --rm -e RABBITMQ_DEFAULT_USER=test -e RABBITMQ_DEFAULT_PASS=test -e RABBITMQ_DEFAULT_VHOST=vhost -p 5672:5672 rabbitmq:3