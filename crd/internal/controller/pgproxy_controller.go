/*
Copyright 2025.

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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	intstr "k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	databasev1alpha1 "github.com/ThomasHerve/migration-crd/api/v1alpha1"
)

// PgProxyReconciler reconciles a PgProxy object
type PgProxyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=database.thomas-herve.fr,resources=pgproxies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.thomas-herve.fr,resources=pgproxies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=database.thomas-herve.fr,resources=pgproxies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PgProxy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *PgProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pgProxy databasev1alpha1.PgProxy
	if err := r.Get(ctx, req.NamespacedName, &pgProxy); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pgProxy.Name + "-svc",
			Namespace: pgProxy.Spec.Namespace,
			Labels:    map[string]string{"app": pgProxy.Name},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Port:       5432,
				TargetPort: intstr.FromInt(int(pgProxy.Spec.PostgresServicePort)),
			}},
			ExternalName:    pgProxy.Spec.PostgresServiceHost,
			SessionAffinity: corev1.ServiceAffinityNone,
		},
	}
	if err := ctrl.SetControllerReference(&pgProxy, svc, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	// Crée ou met à jour
	if err := r.Client.Create(ctx, svc); err != nil {
		if apierrors.IsAlreadyExists(err) {

		} else {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PgProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1alpha1.PgProxy{}).
		Complete(r)
}
