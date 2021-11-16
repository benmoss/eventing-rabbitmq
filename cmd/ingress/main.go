/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/kelseyhightower/envconfig"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"knative.dev/eventing/pkg/kncloudevents"
	"knative.dev/pkg/logging"
)

const (
	defaultMaxIdleConnections        = 1000
	defaultMaxIdleConnectionsPerHost = 1000
)

type envConfig struct {
	Port         int    `envconfig:"PORT" default:"8080"`
	BrokerURL    string `envconfig:"BROKER_URL" required:"true"`
	ExchangeName string `envconfig:"EXCHANGE_NAME" required:"true"`

	channel  *amqp.Channel
	confirms chan amqp.Confirmation
	logger   *zap.SugaredLogger
	counter  int
	mut      sync.Mutex
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatal("Failed to process env var", zap.Error(err))
	}

	conn, err := amqp.Dial(env.BrokerURL)
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %s", err)
	}
	defer conn.Close()

	env.channel, err = conn.Channel()
	if err != nil {
		log.Fatalf("failed to open a channel: %s", err)
	}
	defer env.channel.Close()

	env.confirms = env.channel.NotifyPublish(make(chan amqp.Confirmation))
	env.channel.Confirm(false)

	env.logger = logging.FromContext(context.Background())

	connectionArgs := kncloudevents.ConnectionArgs{
		MaxIdleConns:        defaultMaxIdleConnections,
		MaxIdleConnsPerHost: defaultMaxIdleConnectionsPerHost,
	}
	kncloudevents.ConfigureConnectionArgs(&connectionArgs)

	receiver := kncloudevents.NewHTTPMessageReceiver(env.Port)

	if err := receiver.StartListen(context.Background(), &env); err != nil {
		log.Fatalf("failed to start listen, %v", err)
	}
}

func (env *envConfig) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// validate request method
	if request.Method != http.MethodPost {
		env.logger.Warn("unexpected request method", zap.String("method", request.Method))
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// validate request URI
	if request.RequestURI != "/" {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	ctx := request.Context()

	message := cehttp.NewMessageFromHttpRequest(request)
	defer message.Finish(nil)

	event, err := binding.ToEvent(ctx, message)
	if err != nil {
		env.logger.Warn("failed to extract event from request", zap.Error(err))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	// run validation for the extracted event
	validationErr := event.Validate()
	if validationErr != nil {
		env.logger.Warn("failed to validate extracted event", zap.Error(validationErr))
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	statusCode, err := env.send(event)
	if err != nil {
		env.logger.Error("failed to send event,", err)
	}
	writer.WriteHeader(statusCode)
}

func (env *envConfig) send(event *cloudevents.Event) (int, error) {
	env.mut.Lock()
	defer env.mut.Unlock()

	bytes, err := json.Marshal(event)
	if err != nil {
		env.logger.Info("failed to marshal event")
		return http.StatusBadRequest, fmt.Errorf("failed to marshal event, %w", err)
	}
	env.counter++
	logger := env.logger.With(zap.Int("counter", env.counter))
	logger.Infow("publishing")
	headers := amqp.Table{
		"type":    event.Type(),
		"source":  event.Source(),
		"subject": event.Subject(),
	}
	for key, val := range event.Extensions() {
		headers[key] = val
	}
	if err := env.channel.Publish(
		env.ExchangeName,
		"",    // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			Headers:     headers,
			ContentType: "application/json",
			Body:        bytes,
		}); err != nil {
		logger.Info("failed to publish message")
		return http.StatusInternalServerError, errors.New("failed to publish message")
	}
	logger.Infow("waiting on the confirm")
	confirmed := <-env.confirms
	if confirmed.Ack {
		logger.Infow("ack", zap.Uint64("delivery-tag", confirmed.DeliveryTag))
		return http.StatusAccepted, nil
	} else {
		logger.Infow("nack", zap.Uint64("delivery-tag", confirmed.DeliveryTag))
		return http.StatusServiceUnavailable, errors.New("message was not confirmed")
	}

}
