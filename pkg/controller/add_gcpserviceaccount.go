package controller

import (
	"github.com/kiwigrid/gcp-serviceaccount-controller/pkg/controller/gcpserviceaccount"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, gcpserviceaccount.Add)
}
