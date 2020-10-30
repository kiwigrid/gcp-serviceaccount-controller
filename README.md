# Gcp Service Account Controller

![CI build and Deploy](https://github.com/kiwigrid/gcp-serviceaccount-controller/workflows/CI%20build%20and%20Deploy/badge.svg)

this controller manges gcp service account over kubernetes resources.
    
The Helm chart can be found in the Kiwigrid helm repo. Add it via:

```console
helm repo add kiwigrid https://kiwigrid.github.io
```

The Helm charts source can be found at:

https://github.com/kiwigrid/helm-charts/tree/master/charts/gcp-serviceaccount-controller


## Features

- creates gcp service accounts and creates secrets from the service account keyfile
- handles the full lifecycle of a service account via CRD
- keyfiles are only exists inside kubernetes and not saved outside
- with version 0.2.0 you can restrict enabled roles per namespace via regular expressions (this feature is enabled by default; can be disabled with `DISABLE_RESTRICTION_CHECK`)


## Deployment

First you need to create a GCP service account with at least the following permissions:


```console
- iam.serviceAccounts.create
- iam.serviceAccounts.delete
- iam.serviceAccounts.get
- iam.serviceAccounts.list
- iam.serviceAccounts.update
- iam.serviceAccountKeys.create
- iam.serviceAccountKeys.delete
- iam.serviceAccountKeys.get
- iam.serviceAccountKeys.list
- pubsub.subscriptions.getIamPolicy
- pubsub.subscriptions.setIamPolicy
- pubsub.topics.getIamPolicy
- pubsub.topics.setIamPolicy
- storage.buckets.getIamPolicy
- storage.buckets.setIamPolicy
- resourcemanager.projects.getIamPolicy
- resourcemanager.projects.setIamPolicy
```

You can use the helm chart to deploy
Then add the base64 encoded file to the `gcpCredentials` value.

```console
helm upgrade -i -f <YOUR_VALUES_FILE> <RELEASE_NAME> helm/
```

## Example

This is an example resource definition for a service account:

```yaml
apiVersion: gcp.kiwigrid.com/v1beta1
kind: GcpServiceAccount
metadata:
  name: gcpserviceaccount-sample
spec:
  serviceAccountIdentifier: kube-example
  serviceAccountDescription: kube-example
  secretName: kube-example-secret
  bindings:
  - resource: "//cloudresourcemanager.googleapis.com/projects/<PROJECT_NAME>"
    roles:
    - "roles/cloudsql.editor"
```

Example for buckets:

```yaml
apiVersion: gcp.kiwigrid.com/v1beta1
kind: GcpServiceAccount
metadata:
  name: gcpserviceaccount-bucket-sample
spec:
  serviceAccountIdentifier: kube-bucket-example
  serviceAccountDescription: kube-bucket-example
  secretName: kube-bucket-example-secret
  bindings:
  - resource: buckets/my-bucket-name
    roles:
    - roles/storage.objectAdmin
```

Example for namespace restriction:

```yaml
apiVersion: gcp.kiwigrid.com/v1beta1
kind: GcpNamespaceRestriction
metadata:
  labels:
  name: gcpnamespacerestriction-sample
spec:
  namespace: test
  regex: true
  restrictions:
  - resource: "^buckets/my-bucket-name$"
    roles:
    - "^roles/storage\.objectAdmin$"
  - resource: "^pubsub/.*$"
    roles:
    - "^roles/.*$"
```
