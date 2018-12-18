package gcpserviceaccount

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/go-logr/logr"
	gcpv1beta1 "github.com/kiwigrid/gcp-serviceaccount-controller/pkg/apis/gcp/v1beta1"
	"github.com/kiwigrid/gcp-serviceaccount-controller/pkg/service"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
)

const (
	iamKiwigridFinalizerName = "iam.finalizers.kiwigrid.com"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new GcpServiceAccount Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
// USER ACTION REQUIRED: update cmd/manager/main.go to call this gcp.Add(mgr) to install this Controller
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileGcpServiceAccount{Client: mgr.GetClient(), scheme: mgr.GetScheme(), GcpService: gcpservice.NewGcpService(), log: logf.Log.WithName("gcpserviceaccount-controller")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("gcpserviceaccount-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to GcpServiceAccount
	err = c.Watch(&source.Kind{Type: &gcpv1beta1.GcpServiceAccount{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Uncomment watch a Deployment created by GcpServiceAccount - change this for objects you create
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &gcpv1beta1.GcpServiceAccount{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileGcpServiceAccount{}

// ReconcileGcpServiceAccount reconciles a GcpServiceAccount object
type ReconcileGcpServiceAccount struct {
	client.Client
	scheme *runtime.Scheme
	log    logr.Logger
	*gcpservice.GcpService
}

// Reconcile reads that state of the cluster for a GcpServiceAccount object and makes changes based on the state read
// and what is in the GcpServiceAccount.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gcp.kiwigrid.com,resources=gcpserviceaccounts,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileGcpServiceAccount) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the GcpServiceAccount instance
	instance := &gcpv1beta1.GcpServiceAccount{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			r.log.Info("gcp service account deleted", "name", request.NamespacedName, "err", err)
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

	r.GcpService.HandleAimRoles(instance, "")

	ok, err = r.GcpService.CheckServiceAccountKeyExists(instance, "")

	if err != nil {
		return reconcile.Result{}, err
	}

	found := &corev1.Secret{}
	searchSecretError := r.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.SecretName, Namespace: instance.Namespace}, found)
	deploy := &corev1.Secret{}

	//service account does not exists
	if !ok || (searchSecretError != nil && errors.IsNotFound(searchSecretError)) || (searchSecretError == nil && len(found.Data["credentials.json"]) == 0) {
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
		deploy.Data["credentials.json"] = bytes

		if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
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

func (r *ReconcileGcpServiceAccount) deleteExternalDependency(instance *gcpv1beta1.GcpServiceAccount) error {
	r.log.Info("deleting the external dependencies")

	//TODO add remove impl

	return nil
}

//
// Helper functions to check and remove string from a slice of strings.
//
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
