/*


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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gcpv1beta1 "github.com/kiwigrid/gcp-serviceaccount-controller/api/v1beta1"
)

// GcpNamespaceRestrictionReconciler reconciles a GcpNamespaceRestriction object
type GcpNamespaceRestrictionReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gcp.kiwigrid.com,resources=gcpnamespacerestrictions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gcp.kiwigrid.com,resources=gcpnamespacerestrictions/status,verbs=get;update;patch

func (r *GcpNamespaceRestrictionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("gcpnamespacerestriction", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *GcpNamespaceRestrictionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gcpv1beta1.GcpNamespaceRestriction{}).
		Complete(r)
}
