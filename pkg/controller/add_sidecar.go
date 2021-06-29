package controller

import (
	"github.com/vmware/generic-sidecar-injector/pkg/controller/sidecar"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, sidecar.Add)
}
