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

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1 "knative.dev/eventing/pkg/apis/flows/v1"
)

// SequenceLister helps list Sequences.
// All objects returned here must be treated as read-only.
type SequenceLister interface {
	// List lists all Sequences in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Sequence, err error)
	// Sequences returns an object that can list and get Sequences.
	Sequences(namespace string) SequenceNamespaceLister
	SequenceListerExpansion
}

// sequenceLister implements the SequenceLister interface.
type sequenceLister struct {
	indexer cache.Indexer
}

// NewSequenceLister returns a new SequenceLister.
func NewSequenceLister(indexer cache.Indexer) SequenceLister {
	return &sequenceLister{indexer: indexer}
}

// List lists all Sequences in the indexer.
func (s *sequenceLister) List(selector labels.Selector) (ret []*v1.Sequence, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Sequence))
	})
	return ret, err
}

// Sequences returns an object that can list and get Sequences.
func (s *sequenceLister) Sequences(namespace string) SequenceNamespaceLister {
	return sequenceNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// SequenceNamespaceLister helps list and get Sequences.
// All objects returned here must be treated as read-only.
type SequenceNamespaceLister interface {
	// List lists all Sequences in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.Sequence, err error)
	// Get retrieves the Sequence from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.Sequence, error)
	SequenceNamespaceListerExpansion
}

// sequenceNamespaceLister implements the SequenceNamespaceLister
// interface.
type sequenceNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Sequences in the indexer for a given namespace.
func (s sequenceNamespaceLister) List(selector labels.Selector) (ret []*v1.Sequence, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Sequence))
	})
	return ret, err
}

// Get retrieves the Sequence from the indexer for a given namespace and name.
func (s sequenceNamespaceLister) Get(name string) (*v1.Sequence, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("sequence"), name)
	}
	return obj.(*v1.Sequence), nil
}
