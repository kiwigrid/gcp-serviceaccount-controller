package apis

import (
	"github.com/kiwigrid/gcp-serviceaccount-controller/pkg/apis/gcp/v1beta1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1beta1.SchemeBuilder.AddToScheme)
}
