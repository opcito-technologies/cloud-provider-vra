/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vra

import (
	"k8s.io/klog"
	"context"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/api/core/v1"
)

// Instances encapsulates an implementation of Instances for OpenStack.
type Instances struct {

}

// Instances returns an implementation of Instances for Vra.
func (vr *Vra) Instances() (cloudprovider.Instances, bool) {
	klog.V(4).Info("vra.Instances() called")

	return &Instances{}, true
}


func (i *Instances) CurrentNodeName(ctx context.Context, hostname string) (types.NodeName, error) {

    return types.NodeName("xtest"), nil
}

func (i *Instances) InstanceExistsByProviderID(ctx context.Context, providerID string) (bool, error) {

	return true, nil
}

// AddSSHKeyToAllInstances is not implemented for vra
func (i *Instances) AddSSHKeyToAllInstances(ctx context.Context, user string, keyData []byte) error {
	return cloudprovider.NotImplemented
}

// InstanceID returns the kubelet's cloud provider ID.
func (vr *Vra) InstanceID() (string, error) {
	return "", nil
}

// InstanceID returns the cloud provider ID of the specified instance.
func (i *Instances) InstanceID(ctx context.Context, name types.NodeName) (string, error) {
	return "", nil
}

// InstanceShutdownByProviderID returns true if the instances is in safe state to detach volumes
func (i *Instances) InstanceShutdownByProviderID(ctx context.Context, providerID string) (bool, error) {
	
	return false, nil
}

// InstanceType returns the type of the specified instance.
func (i *Instances) InstanceType(ctx context.Context, name types.NodeName) (string, error) {

	return "", nil
}

func (i *Instances) InstanceTypeByProviderID(ctx context.Context, providerID string) (string, error) {

	return "", nil
}

func (i *Instances) NodeAddressesByProviderID(ctx context.Context, providerID string) ([]v1.NodeAddress, error) {

	return nil, nil
}

// NodeAddresses implements Instances.NodeAddresses
func (i *Instances) NodeAddresses(ctx context.Context, name types.NodeName) ([]v1.NodeAddress, error) {
	klog.V(4).Infof("NodeAddresses(%v) called", name)

	return nil, nil
}


// ExternalID returns the cloud provider ID of the specified instance (deprecated).
func (i *Instances) ExternalID(ctx context.Context, name types.NodeName) (string, error) {
	return "", nil
}

