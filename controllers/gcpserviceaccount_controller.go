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
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gcpv1beta1 "github.com/kiwigrid/gcp-serviceaccount-controller/api/v1beta1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	iamKiwigridFinalizerName = "iam.finalizers.kiwigrid.com"
)

// GcpServiceAccountReconciler reconciles a GcpServiceAccount object
type GcpServiceAccountReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	*GcpService
	RestrictionService  RestrictionService
	DisableRestrictions bool
}

// +kubebuilder:rbac:groups=gcp.kiwigrid.com,resources=gcpserviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gcp.kiwigrid.com,resources=gcpserviceaccounts/status,verbs=get;update;patch

func (r *GcpServiceAccountReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("gcpserviceaccount", request.NamespacedName)
	// Fetch the GcpServiceAccount instance
	instance := &gcpv1beta1.GcpServiceAccount{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			r.log.Info("gcp service account deleted", "name", request.NamespacedName)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object.
		if !containsString(instance.ObjectMeta.Finalizers, iamKiwigridFinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, iamKiwigridFinalizerName)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{Requeue: true}, nil
			}
		}
	} else {
		// The object is being deleted
		if containsString(instance.ObjectMeta.Finalizers, iamKiwigridFinalizerName) {
			// our finalizer is present, so lets handle our external dependency
			if err := r.deleteExternalDependency(instance); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return reconcile.Result{}, err
			}

			// remove our finalizer from the list and update it.
			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, iamKiwigridFinalizerName)
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{Requeue: true}, nil
			}
		}

		// Our finalizer has finished, so the reconciler can do nothing.
		return reconcile.Result{}, nil
	}

	r.log.Info("Start Reconcile", "resourceName", instance.Name)
	if !r.DisableRestrictions {
		hasRights, err := r.RestrictionService.CheckNamespaceHasRights(instance.Namespace, instance.Spec.GcpRoleBindings)
		if err != nil {
			return reconcile.Result{}, err
		}
		if !hasRights {
			return reconcile.Result{}, fmt.Errorf("not enough rights for namespace %s to create serviceaccount for resource %s", instance.Namespace, instance.Name)
		}
	}
	ok, err := r.GcpService.CheckServiceAccountExists(instance, "")
	if err != nil {
		return reconcile.Result{}, err
	}

	if !ok {
		r.log.Info("create new service account")
		account, err := r.GcpService.NewServiceAccount(instance, "")
		if err != nil {
			return reconcile.Result{}, err
		}
		r.log.Info("service account created")
		split := strings.Split(account.Name, "/")
		eMail := split[3]

		instance.Status = gcpv1beta1.GcpServiceAccountStatus{
			ServiceAccountPath: account.Name,
			ServiceAccountMail: eMail,
		}

		err = r.Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	err = r.GcpService.HandleAimRoles(instance, "")
	if err != nil {
		return reconcile.Result{}, err
	}
	instance.Status.AppliedGcpRoleBindings = instance.Spec.GcpRoleBindings

	ok, err = r.GcpService.CheckServiceAccountKeyExists(instance, "")

	if err != nil {
		return reconcile.Result{}, err
	}

	found := &corev1.Secret{}
	searchSecretError := r.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.SecretName, Namespace: instance.Namespace}, found)
	deploy := &corev1.Secret{}

	secretKey := instance.Spec.SecretKey
	if secretKey == "" {
		secretKey = "credentials.json"
	}

	//service account does not exists
	if !ok || (searchSecretError != nil && errors.IsNotFound(searchSecretError)) || (searchSecretError == nil && len(found.Data[secretKey]) == 0) {
		r.log.Info(fmt.Sprintf("create or update secret: %s", instance.Spec.SecretName))
		key, err := r.GcpService.CreateServiceAccountKey(instance, "")
		if err != nil {
			return reconcile.Result{}, err
		}
		instance.Status.CredentialKey = key.Name
		err = r.Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		r.log.Info(fmt.Sprintf("modify secret %s with gcp key %s", instance.Spec.SecretName, key.Name))

		deploy.Name = instance.Spec.SecretName
		deploy.Namespace = instance.Namespace
		deploy.Data = map[string][]byte{}
		bytes, err := base64.StdEncoding.DecodeString(key.PrivateKeyData)
		deploy.Data[secretKey] = bytes

		if err := controllerutil.SetControllerReference(instance, deploy, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}

		//new secret
		if searchSecretError != nil && errors.IsNotFound(searchSecretError) {
			r.log.Info("Creating Secret", "secretName", instance.Spec.SecretName, "namespace", instance.Namespace)
			err = r.Create(context.TODO(), deploy)
			if err != nil {
				return reconcile.Result{}, searchSecretError
			}
			return reconcile.Result{}, nil
		} else if searchSecretError != nil {
			return reconcile.Result{}, searchSecretError
		}

		// Update the found object and write the result back if there are any changes
		if !reflect.DeepEqual(deploy.Data, found.Data) {
			found.Data = deploy.Data
			r.log.Info("Updating Deployment", "namespace", deploy.Namespace, "name", deploy.Name)
			err = r.Update(context.TODO(), found)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	err = r.Update(context.TODO(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *GcpServiceAccountReconciler) deleteExternalDependency(instance *gcpv1beta1.GcpServiceAccount) error {
	r.log.Info("deleting the external dependencies")
	return r.GcpService.DeleteServiceAccount(instance)
}

func (r *GcpServiceAccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gcpv1beta1.GcpServiceAccount{}).
		Complete(r)
}
