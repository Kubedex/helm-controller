package controller

import (
	"gitlab.com/pearsontechnology/helm-controller/pkg/controller/helmchart"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, helmchart.Add)
}
