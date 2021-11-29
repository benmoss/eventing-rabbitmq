/*
Copyright 2021 The Knative Authors

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

package e2e

import (
	"knative.dev/eventing-rabbitmq/test/e2e/config/broker"

	"knative.dev/reconciler-test/pkg/feature"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "knative.dev/pkg/system/testing"
)

// MultipleBrokersUsingSingleRabbitMQClusterTest checks that Brokers in different namespaces can use the same RabbitMQ cluster
func NamespacedBrokerTest(namespace string) *feature.Feature {
	f := new(feature.Feature)

	f.Setup("install Broker in different namespace", broker.Install(namespace))
	f.Alpha("Broker and all dependencies - Exchange, Queue & Binding -").Must("be ready", AllGoReady)
	f.Teardown("delete Broker namespace and all objects", broker.Uninstall(namespace))

	return f
}
