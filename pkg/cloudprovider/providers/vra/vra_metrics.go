/*
Copyright 2017 The Kubernetes Authors.

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
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog"
)

const (
	vraSubsystem         = "vra"
	vraOperationKey      = "cloudprovider_vra_api_request_duration_seconds"
	vraOperationErrorKey = "cloudprovider_vra_api_request_errors"
)

var (
	vraOperationsLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: vraSubsystem,
			Name:      vraOperationKey,
			Help:      "Latency of vra api call",
		},
		[]string{"request"},
	)


	vraAPIRequestErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: vraSubsystem,
			Name:      vraOperationErrorKey,
			Help:      "Cumulative number of vra Api call errors",
		},
		[]string{"request"},
	)
)

func RegisterMetrics() {
	if err := prometheus.Register(vraOperationsLatency); err != nil {
		klog.V(5).Infof("unable to register for latency metrics")
	}
	if err := prometheus.Register(vraAPIRequestErrors); err != nil {
		klog.V(5).Infof("unable to register for error metrics")
	}
}
