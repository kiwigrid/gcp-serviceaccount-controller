package gcpservice

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/kiwigrid/gcp-serviceaccount-controller/pkg/apis/gcp/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type RestrictionResolveService interface {
	CheckNamespaceHasRights(namespace string) (*v1beta1.GcpNamespaceRestriction, error)
}

type RestrictionResolveServiceImpl struct {
	log logr.Logger
	client.Client
}

func NewRestrictionResolveService(kubernetesClient *client.Client) *RestrictionResolveServiceImpl {

	return &RestrictionResolveServiceImpl{
		log:    logf.Log.WithName("restrictionresolveservice"),
		Client: *kubernetesClient}

}

func (r *RestrictionResolveServiceImpl) CheckNamespaceHasRights(namespace string) (*v1beta1.GcpNamespaceRestriction, error) {
	list := &v1beta1.GcpNamespaceRestrictionList{}
	err := r.List(context.TODO(), &client.ListOptions{}, list)
	if err != nil {
		return nil, err
	}
	return findItem(list.Items, namespace), nil
}
