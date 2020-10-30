package controllers

import (
	"github.com/go-logr/logr"
	"github.com/kiwigrid/gcp-serviceaccount-controller/api/v1beta1"
	"regexp"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type RestrictionService struct {
	log            logr.Logger
	resolveService RestrictionResolveService
}

func NewRestrictionService(restrictionResolveService RestrictionResolveService) *RestrictionService {
	return &RestrictionService{
		log:            logf.Log.WithName("restrictionservice"),
		resolveService: restrictionResolveService}
}

func (r *RestrictionService) CheckNamespaceHasRights(namespace string, resources []v1beta1.GcpRoleBindings) (bool, error) {
	restriction, err := r.resolveService.CheckNamespaceHasRights(namespace)
	if err != nil {
		return false, err
	}
	//check if we find a match for each entry
	for _, res := range resources {
		if res.Resource != "" {
			find, binding := r.getMatchingResource(restriction, res.Resource)
			if !find {
				return false, nil
			}
			return r.checkAllRolesMatch(binding, res.Roles, restriction.Spec.Regex), nil
		}

	}
	return false, nil
}

func (r *RestrictionService) checkAllRolesMatch(binding *v1beta1.GcpRestrictionRoleBinding, roles []string, regex bool) bool {
	for _, role := range roles {
		found := false
		for _, check := range binding.Roles {
			if r.matches(check, role, regex) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (r *RestrictionService) matches(check string, toCheck string, regex bool) bool {
	if regex {
		r, err := regexp.Compile(check)
		if err != nil {
			return false
		}
		return r.MatchString(toCheck)
	} else {
		return check == toCheck
	}
}

func (r *RestrictionService) getMatchingResource(restriction *v1beta1.GcpNamespaceRestriction, resource string) (bool, *v1beta1.GcpRestrictionRoleBinding) {
	for _, res := range restriction.Spec.GcpRestriction {
		if res.Resource != "" && r.matches(res.Resource, resource, restriction.Spec.Regex) {
			return true, &res
		}
	}
	return false, nil
}

func findItem(list []v1beta1.GcpNamespaceRestriction, namespace string) *v1beta1.GcpNamespaceRestriction {
	for _, element := range list {
		if element.Spec.Namespace == namespace {
			return &element
		}
	}
	return nil
}
