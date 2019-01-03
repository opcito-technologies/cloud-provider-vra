
package vra

import (
	"context"
	"fmt"
	"regexp"

	"k8s.io/klog"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
)

// Instances encapsulates an implementation of Instances for Vra.
type Instances struct {

}

const (
	instanceShutoff = "SHUTOFF"
)

// Instances returns an implementation of Instances for Vra.
func (vra *Vra) Instances() (cloudprovider.Instances, bool) {
	klog.V(4).Info("vra.Instances() called")

	compute, err := vra.NewComputeV2()
	if err != nil {
		klog.Errorf("unable to access machine API : %v", err)
		return nil, false
	}

	klog.V(4).Info("Claiming to support Instances")

}
