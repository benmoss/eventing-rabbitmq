package v1alpha2

import (
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func (p *Policy) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(p).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-rabbitmq-com-v1alpha2-policy,mutating=false,failurePolicy=fail,groups=rabbitmq.com,resources=policies,versions=v1alpha2,name=vpolicy.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Policy{}

// no validation on create
func (p *Policy) ValidateCreate() error {
	return nil
}

// returns error type 'forbidden' for updates on policy name, vhost and rabbitmqClusterReference
func (p *Policy) ValidateUpdate(old runtime.Object) error {
	oldPolicy, ok := old.(*Policy)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a policy but got a %T", old))
	}

	detailMsg := "updates on name, vhost and rabbitmqClusterReference are all forbidden"
	if p.Spec.Name != oldPolicy.Spec.Name {
		return apierrors.NewForbidden(p.GroupResource(), p.Name,
			field.Forbidden(field.NewPath("spec", "name"), detailMsg))
	}

	if p.Spec.Vhost != oldPolicy.Spec.Vhost {
		return apierrors.NewForbidden(p.GroupResource(), p.Name,
			field.Forbidden(field.NewPath("spec", "vhost"), detailMsg))
	}

	if p.Spec.RabbitmqClusterReference != oldPolicy.Spec.RabbitmqClusterReference {
		return apierrors.NewForbidden(p.GroupResource(), p.Name,
			field.Forbidden(field.NewPath("spec", "rabbitmqClusterReference"), detailMsg))
	}
	return nil
}

// no validation on delete
func (p *Policy) ValidateDelete() error {
	return nil
}
