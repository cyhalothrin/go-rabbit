package rabbit

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Enveloper common interface for events or RPC call, it allows you to create your own message type with custom logic
// for handling some fields of amqp091.Publishing
type Enveloper interface {
	Envelop() (*Envelop, error)
}

type Envelop struct {
	Payload   Publishing
	Exchange  string
	Key       string
	Mandatory bool
	Immediate bool
}

func (e *Envelop) Envelop() (*Envelop, error) {
	return e, nil
}

type JSONEnvelop struct {
	Payload   JSONPublishing
	Exchange  string
	Key       string
	Mandatory bool
	Immediate bool
}

func (j *JSONEnvelop) Envelop() (Envelop, error) {
	body, err := json.Marshal(j.Payload.Data)
	if err != nil {
		return Envelop{}, err
	}

	pub := j.Payload.Publishing
	pub.Body = body

	return Envelop{
			Payload:   pub,
			Exchange:  j.Exchange,
			Key:       j.Key,
			Mandatory: j.Mandatory,
			Immediate: j.Immediate,
		},
		nil
}

type JSONPublishing struct {
	amqp.Publishing
	Data interface{}
}
