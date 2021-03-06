package controllers

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/go-logr/logr"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-gcp-common/gcputil"
	"github.com/hashicorp/vault-plugin-secrets-gcp/plugin/iamutil"
	"github.com/hashicorp/vault-plugin-secrets-gcp/plugin/util"
	"github.com/hashicorp/vault/sdk/helper/useragent"
	gcpv1beta1 "github.com/kiwigrid/gcp-serviceaccount-controller/api/v1beta1"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iam/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	serviceAccountMaxLen          = 30
	serviceAccountDisplayNameTmpl = "Service account for Vault secrets backend role set %s"
	defaultCloudPlatformScope     = "https://www.googleapis.com/auth/cloud-platform"
	keyAlgorithmRSA2k             = "KEY_ALG_RSA_2048"
	privateKeyTypeJson            = "TYPE_GOOGLE_CREDENTIALS_FILE"
)

type GcpService struct {
	log      logr.Logger
	iamAdmin *iam.Service
}

func NewGcpService() *GcpService {
	service, err := newIamAdmin(context.TODO())
	if err == nil {
		return &GcpService{log: logf.Log.WithName("gcpservice"), iamAdmin: service}
	}
	return nil
}

func (s *GcpService) CheckServiceAccountExists(gcpServiceAccount *gcpv1beta1.GcpServiceAccount, project string) (bool, error) {
	if project == "" {
		gcpCred, _, err := gcputil.FindCredentials("", context.TODO(), defaultCloudPlatformScope)
		if err != nil {
			return false, err
		}
		if gcpCred == nil {
			return false, fmt.Errorf("error finding gcp credentials file")
		}
		project = gcpCred.ProjectId
	}

	_, err := s.iamAdmin.Projects.ServiceAccounts.Get(gcpServiceAccount.Status.ServiceAccountPath).Do()
	if err != nil {
		e, ok := err.(*googleapi.Error)
		if !ok {
			return false, err
		}
		if e.Code == 404 {
			return false, nil
		}
	}
	return true, nil
}
func (s *GcpService) CheckServiceAccountKeyExists(gcpServiceAccount *gcpv1beta1.GcpServiceAccount, project string) (bool, error) {
	if project == "" {
		gcpCred, _, _ := gcputil.FindCredentials("", context.TODO(), defaultCloudPlatformScope)
		project = gcpCred.ProjectId
	}
	_, err := s.iamAdmin.Projects.ServiceAccounts.Keys.Get(gcpServiceAccount.Status.CredentialKey).Do()

	if err != nil {
		e, ok := err.(*googleapi.Error)
		if !ok {
			return false, err
		}
		if e.Code == 404 {
			return false, nil
		}
	}
	return true, nil
}

func (s *GcpService) CreateServiceAccountKey(gcpServiceAccount *gcpv1beta1.GcpServiceAccount, project string) (*iam.ServiceAccountKey, error) {
	response, err := s.iamAdmin.Projects.ServiceAccounts.Keys.List(gcpServiceAccount.Status.ServiceAccountPath).KeyTypes("USER_MANAGED").Do()
	if err != nil && !isGoogleApi404Error(err) {
		return nil, errwrap.Wrapf(fmt.Sprintf("unable to listservice account key for service account '%s': {{err}}", gcpServiceAccount.Status.ServiceAccountPath), err)
	}
	if response.Keys != nil && len(response.Keys) > 0 {
		for _, k := range response.Keys {
			_, err = s.iamAdmin.Projects.ServiceAccounts.Keys.Delete(k.Name).Do()
			if err != nil {
				return nil, errwrap.Wrapf(fmt.Sprintf("unable to delete service account key %s for service account '%s': {{err}}", k.Name, gcpServiceAccount.Status.ServiceAccountPath), err)
			}
		}
	}

	key, err := s.iamAdmin.Projects.ServiceAccounts.Keys.Create(gcpServiceAccount.Status.ServiceAccountPath,
		&iam.CreateServiceAccountKeyRequest{
			PrivateKeyType: privateKeyTypeJson,
		}).Do()
	if err != nil {
		return nil, errwrap.Wrapf(fmt.Sprintf("unable to create new service account key for service account '%s': {{err}}", gcpServiceAccount.Status.ServiceAccountPath), err)
	}
	return key, nil
}

func (s *GcpService) HandleAimRoles(gcpServiceAccount *gcpv1beta1.GcpServiceAccount, project string) error {

	if project == "" {
		gcpCred, _, _ := gcputil.FindCredentials("", context.TODO(), defaultCloudPlatformScope)
		project = gcpCred.ProjectId
	}

	iamResources := iamutil.GetEnabledResources()
	httpC, err := newHttpClient(context.TODO(), defaultCloudPlatformScope)
	if err != nil {
		return err
	}

	iamHandle := iamutil.GetApiHandle(httpC, useragent.String())

	for _, bindings := range gcpServiceAccount.Status.AppliedGcpRoleBindings {

		resource, err := iamResources.Parse(bindings.Resource)
		if err != nil {
			return err
		}

		p, err := resource.GetIamPolicy(context.TODO(), iamHandle)
		if err != nil {
			return err
		}

		roles := util.StringSet{}
		for _, role := range bindings.Roles {
			roles.Add(role)
		}

		changed, newP := p.RemoveBindings(&iamutil.PolicyDelta{
			Roles: roles,
			Email: gcpServiceAccount.Status.ServiceAccountMail,
		})
		if !changed || newP == nil {
			s.log.Info("role binding not changed skip", "resource", bindings.Resource)
			continue
		}
		if _, err := resource.SetIamPolicy(context.TODO(), iamHandle, newP); err != nil {
			return err
		}

	}

	for _, bindings := range gcpServiceAccount.Spec.GcpRoleBindings {

		resource, err := iamResources.Parse(bindings.Resource)
		if err != nil {
			return err
		}

		p, err := resource.GetIamPolicy(context.TODO(), iamHandle)
		if err != nil {
			return err
		}

		roles := util.StringSet{}
		for _, role := range bindings.Roles {
			roles.Add(role)
		}

		changed, newP := p.AddBindings(&iamutil.PolicyDelta{
			Roles: roles,
			Email: gcpServiceAccount.Status.ServiceAccountMail,
		})
		if !changed || newP == nil {
			s.log.Info("role binding not changed skip", "resource", bindings.Resource)
			continue
		}

		if _, err := resource.SetIamPolicy(context.TODO(), iamHandle, newP); err != nil {
			return err
		}

	}
	return nil
}

func (s *GcpService) NewServiceAccount(gcpServiceAccount *gcpv1beta1.GcpServiceAccount, project string) (*iam.ServiceAccount, error) {
	if project == "" {
		gcpCred, _, _ := gcputil.FindCredentials("", context.TODO(), defaultCloudPlatformScope)
		project = gcpCred.ProjectId
	}

	saEmailPrefix := roleSetServiceAccountName(gcpServiceAccount.Spec.ServiceAccountIdentifier)
	projectName := fmt.Sprintf("projects/%s", project)
	displayName := gcpServiceAccount.Spec.ServiceAccountDescription

	sa, err := s.iamAdmin.Projects.ServiceAccounts.Create(
		projectName, &iam.CreateServiceAccountRequest{
			AccountId:      saEmailPrefix,
			ServiceAccount: &iam.ServiceAccount{DisplayName: displayName},
		}).Do()

	if err != nil {
		return nil, errwrap.Wrapf(fmt.Sprintf("unable to create new service account under project '%s': {{err}}", projectName), err)
	}

	return sa, nil
}

func (s *GcpService) DeleteServiceAccount(account *gcpv1beta1.GcpServiceAccount) error {
	err := s.removeAimRoleBindings(account, "")
	if err != nil {
		return err
	}
	_, err = s.iamAdmin.Projects.ServiceAccounts.Delete(account.Status.ServiceAccountPath).Do()
	if err != nil && !isGoogleApi404Error(err) {
		return err
	}
	return nil
}

func (s *GcpService) removeAimRoleBindings(gcpServiceAccount *gcpv1beta1.GcpServiceAccount, project string) error {

	if project == "" {
		gcpCred, _, _ := gcputil.FindCredentials("", context.TODO(), defaultCloudPlatformScope)
		project = gcpCred.ProjectId
	}

	iamResources := iamutil.GetEnabledResources()
	httpC, err := newHttpClient(context.TODO(), defaultCloudPlatformScope)
	if err != nil {
		return err
	}

	iamHandle := iamutil.GetApiHandle(httpC, useragent.String())

	for _, bindings := range gcpServiceAccount.Status.AppliedGcpRoleBindings {

		resource, err := iamResources.Parse(bindings.Resource)
		if err != nil {
			return err
		}

		p, err := resource.GetIamPolicy(context.TODO(), iamHandle)
		if err != nil {
			return err
		}

		roles := util.StringSet{}
		for _, role := range bindings.Roles {
			roles.Add(role)
		}

		changed, newP := p.RemoveBindings(&iamutil.PolicyDelta{
			Roles: roles,
			Email: gcpServiceAccount.Status.ServiceAccountMail,
		})
		if !changed || newP == nil {
			continue
		}

		if _, err := resource.SetIamPolicy(context.TODO(), iamHandle, newP); err != nil {
			return err
		}

	}
	return nil
}

func newHttpClient(ctx context.Context, scopes ...string) (*http.Client, error) {
	if len(scopes) == 0 {
		scopes = []string{"https://www.googleapis.com/auth/cloud-platform"}
	}

	_, tokenSource, err := gcputil.FindCredentials("", ctx, scopes...)
	if err != nil {
		return nil, err
	}

	tc := cleanhttp.DefaultClient()
	return oauth2.NewClient(
		context.WithValue(ctx, oauth2.HTTPClient, tc),
		tokenSource), nil
}

func newIamAdmin(ctx context.Context) (*iam.Service, error) {
	c, err := newHttpClient(ctx, iam.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	return iam.New(c)
}

func roleSetServiceAccountName(rsName string) (name string) {
	// Sanitize role name
	reg := regexp.MustCompile("[^a-zA-Z0-9-]+")
	rsName = reg.ReplaceAllString(rsName, "-")

	intSuffix := fmt.Sprintf("%d", time.Now().Unix())
	fullName := fmt.Sprintf("kube%s-%s", rsName, intSuffix)
	name = fullName
	if len(fullName) > serviceAccountMaxLen {
		toTrunc := len(fullName) - serviceAccountMaxLen
		name = fmt.Sprintf("kube%s-%s", rsName[:len(rsName)-toTrunc], intSuffix)
	}
	return name
}

func isGoogleApi404Error(err error) bool {
	if err == nil {
		return false
	}
	gErr, ok := err.(*googleapi.Error)
	if ok && gErr.Code == 404 {
		return true
	}
	return false
}
