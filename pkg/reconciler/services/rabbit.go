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

package services

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
	"knative.dev/eventing-rabbitmq/pkg/reconciler/broker/resources"
	triggerresources "knative.dev/eventing-rabbitmq/pkg/reconciler/trigger/resources"
	"knative.dev/eventing-rabbitmq/third_party/pkg/apis/rabbitmq.com/v1beta1"
	rabbitclientset "knative.dev/eventing-rabbitmq/third_party/pkg/client/clientset/versioned"
	versioned "knative.dev/eventing-rabbitmq/third_party/pkg/client/clientset/versioned"
	factory "knative.dev/eventing-rabbitmq/third_party/pkg/client/injection/informers/factory"
	rabbitlisters "knative.dev/eventing-rabbitmq/third_party/pkg/client/listers/rabbitmq.com/v1beta1"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
)

const (
	messagingGroupVersion = "rabbitmq.com/v1beta1"
	messagingExchangeKind = "Exchange"
)

type RabbitService interface {
	ReconcileExchange(context.Context, *resources.ExchangeArgs) (*v1beta1.Exchange, error)
	ReconcileQueue(context.Context, *triggerresources.QueueArgs) (*v1beta1.Queue, error)
	ReconcileBinding(context.Context, *triggerresources.BindingArgs) (*v1beta1.Binding, error)
}

func NewRabbit(ctx context.Context) RabbitService {
	cfg := injection.GetConfig(ctx)

	var useCRDs bool
	client := discovery.NewDiscoveryClientForConfigOrDie(cfg)
	resources, err := client.ServerResourcesForGroupVersion(messagingGroupVersion)
	if errors.IsNotFound(err) {
		logging.FromContext(ctx).Info("rabbitmq resource group not found, defaulting to standalone mode")
		return NewStandaloneRabbit(ctx)
	} else if errors.IsForbidden(err) {
		logging.FromContext(ctx).Error("unable to view installed resources")
		panic(err)
	} else if err != nil {
		logging.FromContext(ctx).Errorw("unknown error", zap.Error(err))
		panic(err)
	}
	for _, resource := range resources.APIResources {
		if resource.Kind == messagingExchangeKind {
			useCRDs = true
		}
	}
	if useCRDs {
		return NewManagedRabbit(ctx, cfg)
	} else {
		return NewStandaloneRabbit(ctx)
	}
}

func NewManagedRabbit(ctx context.Context, cfg *restclient.Config) *ManagedRabbit {
	rabbitClientSet := versioned.NewForConfigOrDie(cfg)
	rabbit := factory.Get(ctx).Rabbitmq().V1beta1()

	return &ManagedRabbit{
		rabbitClientSet: rabbitClientSet,
		exchangeLister:  rabbit.Exchanges().Lister(),
		queueLister:     rabbit.Queues().Lister(),
		bindingLister:   rabbit.Bindings().Lister(),
	}
}

func NewStandaloneRabbit(ctx context.Context) *StandaloneRabbit {
	return nil
}

func NewRabbitTest(
	rabbitClientSet rabbitclientset.Interface,
	exchangeLister rabbitlisters.ExchangeLister,
	queueLister rabbitlisters.QueueLister,
	bindingLister rabbitlisters.BindingLister,
) *ManagedRabbit {
	// TODO(bmo): remove, use mocks
	return &ManagedRabbit{
		rabbitClientSet: rabbitClientSet,
		exchangeLister:  exchangeLister,
		queueLister:     queueLister,
		bindingLister:   bindingLister,
	}
}

type ManagedRabbit struct {
	rabbitClientSet rabbitclientset.Interface
	exchangeLister  rabbitlisters.ExchangeLister
	queueLister     rabbitlisters.QueueLister
	bindingLister   rabbitlisters.BindingLister
}

func (r *ManagedRabbit) ReconcileExchange(ctx context.Context, args *resources.ExchangeArgs) (*v1beta1.Exchange, error) {
	logging.FromContext(ctx).Infow("Reconciling exchange", zap.String("name", args.Name))

	want := resources.NewExchange(ctx, args)
	current, err := r.exchangeLister.Exchanges(args.Broker.Namespace).Get(args.Name)
	if apierrs.IsNotFound(err) {
		logging.FromContext(ctx).Debugw("Creating rabbitmq exchange", zap.String("exchange name", want.Name))
		return r.rabbitClientSet.RabbitmqV1beta1().Exchanges(args.Broker.Namespace).Create(ctx, want, metav1.CreateOptions{})
	} else if err != nil {
		return nil, err
	} else if !equality.Semantic.DeepDerivative(want.Spec, current.Spec) {
		// Don't modify the informers copy.
		desired := current.DeepCopy()
		desired.Spec = want.Spec
		return r.rabbitClientSet.RabbitmqV1beta1().Exchanges(args.Broker.Namespace).Update(ctx, desired, metav1.UpdateOptions{})
	}
	return current, nil
}

func (r *ManagedRabbit) ReconcileQueue(ctx context.Context, args *triggerresources.QueueArgs) (*v1beta1.Queue, error) {
	logging.FromContext(ctx).Info("Reconciling queue")

	queueName := args.Name
	want := triggerresources.NewQueue(ctx, args)
	current, err := r.queueLister.Queues(args.Namespace).Get(queueName)
	if apierrs.IsNotFound(err) {
		logging.FromContext(ctx).Debugw("Creating rabbitmq exchange", zap.String("queue name", want.Name))
		return r.rabbitClientSet.RabbitmqV1beta1().Queues(args.Namespace).Create(ctx, want, metav1.CreateOptions{})
	} else if err != nil {
		return nil, err
	} else if !equality.Semantic.DeepDerivative(want.Spec, current.Spec) {
		// Don't modify the informers copy.
		desired := current.DeepCopy()
		desired.Spec = want.Spec
		return r.rabbitClientSet.RabbitmqV1beta1().Queues(args.Namespace).Update(ctx, desired, metav1.UpdateOptions{})
	}
	return current, nil
}

func (r *ManagedRabbit) ReconcileBinding(ctx context.Context, args *triggerresources.BindingArgs) (*v1beta1.Binding, error) {
	logging.FromContext(ctx).Info("Reconciling binding")

	want, err := triggerresources.NewBinding(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to create the binding spec: %w", err)
	}
	current, err := r.bindingLister.Bindings(args.Namespace).Get(args.Name)
	if apierrs.IsNotFound(err) {
		logging.FromContext(ctx).Infow("Creating rabbitmq binding", zap.String("binding name", want.Name))
		return r.rabbitClientSet.RabbitmqV1beta1().Bindings(args.Namespace).Create(ctx, want, metav1.CreateOptions{})
	} else if err != nil {
		return nil, err
	} else if !equality.Semantic.DeepDerivative(want.Spec, current.Spec) {
		// Don't modify the informers copy.
		desired := current.DeepCopy()
		desired.Spec = want.Spec
		return r.rabbitClientSet.RabbitmqV1beta1().Bindings(args.Namespace).Update(ctx, desired, metav1.UpdateOptions{})
	}
	return current, nil
}

type StandaloneRabbit struct {
}

// TODO: make it work?
func (r *StandaloneRabbit) ReconcileExchange(ctx context.Context, args *resources.ExchangeArgs) (*v1beta1.Exchange, error) {
	return nil, nil
}

func (r *StandaloneRabbit) ReconcileQueue(ctx context.Context, args *triggerresources.QueueArgs) (*v1beta1.Queue, error) {
	return nil, nil
}

func (r *StandaloneRabbit) ReconcileBinding(ctx context.Context, args *triggerresources.BindingArgs) (*v1beta1.Binding, error) {
	return nil, nil
}
