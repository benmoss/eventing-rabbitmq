package hack

import (
	"github.com/cloudevents/sdk-go/v2/event"
	amqp "github.com/rabbitmq/amqp091-go"
	rabbitmq "knative.dev/eventing-rabbitmq/pkg/adapter"
)

// this sorta exists already in pkg/amqp/amqp.go
type DialerFunc func(url string) (Broker, error)

type ConsumeArgs struct {
	QueueName     string
	PrefetchCount int
}

type QueueBindArgs struct {
	Name,
	Key,
	Source string
	NoWait bool
}

// interfaces for a hypothetical backing message broker (Rabbit, in-memory fake for testing)
type Channel interface {
	// handles converting messages into cloudevents! hides static config, autoack=false etc
	Consume(args ConsumeArgs) (<-chan event.Event, error)

	// declare a queue with this struct we already have that almost has all the fields we need
	QueueDeclare(args rabbitmq.QueueConfig) (amqp.Queue, error)

	// declare an exchange
	ExchangeDeclare(args rabbitmq.ExchangeConfig) error

	// bind a QueueDeclare
	QueueBind(args QueueBindArgs) error

	// delete a queue
	QueueDelete(name string) error

	// delete an exchange
	ExchangeDelete(name string) error
}

/*
dispatcher:
- cmd/dispatcher/main.go
- pkg/dispatcher/dispatcher.go
connect to rabbit, establish a channel, set qos, consume from channel

adapter:
- cmd/adapter/adapter.go
- pkg/adapter/adapter.go
connect to rabbit, establish a channel, set qos, declare exchange, declare queue, bind queue, consume from channel

standalone controller:
- pkg/reconciler/triggerstandalone/resources/queue.go
- pkg/reconciler/triggerstandalone/resources/exchange.go
connect to rabbit, declare a queue, delete a queue
connect to rabbit, declare an exchange, delete an exchange
*/
