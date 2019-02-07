package gcpservice

import (
	"github.com/kiwigrid/gcp-serviceaccount-controller/pkg/apis/gcp/v1beta1"
	"testing"
)

type RestrictionResolveServiceFake struct {
	Fake *v1beta1.GcpNamespaceRestriction
}

func (r RestrictionResolveServiceFake) CheckNamespaceHasRights(namespace string) (*v1beta1.GcpNamespaceRestriction, error) {
	return r.Fake, nil
}

func TestSimpleRegexMatch(t *testing.T) {

	resolveService := RestrictionResolveServiceFake{
		Fake: &v1beta1.GcpNamespaceRestriction{
			Spec: v1beta1.GcpNamespaceRestrictionSpec{
				Namespace: "test",
				Regex:     true,
				GcpRestriction: []v1beta1.GcpRestrictionRoleBinding{
					v1beta1.GcpRestrictionRoleBinding{
						Resource: "buckets.*",
						Roles:    []string{"roles/storage.*"},
					},
				},
			},
		},
	}
	service := NewRestrictionService(resolveService)

	a := []v1beta1.GcpRoleBindings{
		v1beta1.GcpRoleBindings{
			Resource: "buckets/my-bucket-name",
			Roles: []string{
				"roles/storage.objectAdmin",
			},
		},
	}
	hasRight, err := service.CheckNamespaceHasRights("test", a)

	if err != nil {
		t.Error("expect err to nil")
	}

	if !hasRight {
		t.Error("expect hasRight to be true was false")
	}
}

func TestMultiRoleMultiResourceRegexMatch(t *testing.T) {

	resolveService := RestrictionResolveServiceFake{
		Fake: &v1beta1.GcpNamespaceRestriction{
			Spec: v1beta1.GcpNamespaceRestrictionSpec{
				Namespace: "test",
				Regex:     true,
				GcpRestriction: []v1beta1.GcpRestrictionRoleBinding{
					v1beta1.GcpRestrictionRoleBinding{
						Resource: "^buckets.*$",
						Roles:    []string{"roles/storage.*"},
					},
				},
			},
		},
	}
	service := NewRestrictionService(resolveService)

	a := []v1beta1.GcpRoleBindings{
		v1beta1.GcpRoleBindings{
			Resource: "buckets/my-bucket-name",
			Roles: []string{
				"roles/storage.objectAdmin",
				"roles/storage.objectAdmin1",
				"roles/storage.objectAdmin3",
			},
		},
		v1beta1.GcpRoleBindings{
			Resource: "buckets/my-bucket-name1",
			Roles: []string{
				"roles/storage.objectAdmin",
				"roles/storage.objectAdmin1",
				"roles/storage.objectAdmin3",
			},
		},
	}
	hasRight, err := service.CheckNamespaceHasRights("test", a)

	if err != nil {
		t.Error("expect err to nil")
	}

	if !hasRight {
		t.Error("expect hasRight to be true was false")
	}
}

func TestNoRegex(t *testing.T) {
	resolveService := RestrictionResolveServiceFake{
		Fake: &v1beta1.GcpNamespaceRestriction{
			Spec: v1beta1.GcpNamespaceRestrictionSpec{
				Namespace: "test",
				Regex:     false,
				GcpRestriction: []v1beta1.GcpRestrictionRoleBinding{
					v1beta1.GcpRestrictionRoleBinding{
						Resource: "buckets/my-bucket-name",
						Roles: []string{
							"roles/storage.objectAdmin",
							"roles/xyz"},
					},
				},
			},
		},
	}
	service := NewRestrictionService(resolveService)

	a := []v1beta1.GcpRoleBindings{
		v1beta1.GcpRoleBindings{
			Resource: "buckets/my-bucket-name",
			Roles: []string{
				"roles/storage.objectAdmin",
			},
		},
	}
	hasRight, err := service.CheckNamespaceHasRights("test", a)

	if err != nil {
		t.Error("expect err to nil")
	}

	if !hasRight {
		t.Error("expect hasRight to be true was false")
	}
}
