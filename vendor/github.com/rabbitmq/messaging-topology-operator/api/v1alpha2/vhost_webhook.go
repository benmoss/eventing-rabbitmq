package v1alpha2

import (
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func (r *Vhost) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-rabbitmq-com-v1alpha2-vhost,mutating=false,failurePolicy=fail,groups=rabbitmq.com,resources=vhosts,versions=v1alpha2,name=vvhost.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Vhost{}

// no validation on create
func (v *Vhost) ValidateCreate() error {
	return nil
}

// returns error type 'forbidden' for updates on vhost name and rabbitmqClusterReference
// vhost.spec.tracing can be updated
func (v *Vhost) ValidateUpdate(old runtime.Object) error {
	oldVhost, ok := old.(*Vhost)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a vhost but got a %T", old))
	}

	detailMsg := "updates on name and rabbitmqClusterReference are all forbidden"
	if v.Spec.Name != oldVhost.Spec.Name {
		return apierrors.NewForbidden(v.GroupResource(), v.Name,
			field.Forbidden(field.NewPath("spec", "name"), detailMsg))
	}

	if v.Spec.RabbitmqClusterReference != oldVhost.Spec.RabbitmqClusterReference {
		return apierrors.NewForbidden(v.GroupResource(), v.Name,
			field.Forbidden(field.NewPath("spec", "rabbitmqClusterReference"), detailMsg))
	}

	return nil
}

// no validation on delete
func (v *Vhost) ValidateDelete() error {
	return nil
}
