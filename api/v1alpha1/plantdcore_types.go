/*
Copyright 2023.

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

package v1alpha1

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeploymentConfig defines the desired state of modules managed as Deployment
type DeploymentConfig struct {
	// Image defines the container image to use
	Image string `json:"image,omitempty"`
	// Replicas defines the desired number of replicas
	Replicas int32 `json:"replicas,omitempty"`
	// Resources defines the resource requirements per replica
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// PrometheusConfig defines the desired state of Prometheus
type PrometheusConfig struct {
	// ScrapeInterval defines the desired time length between scrapings
	ScrapeInterval monitoringv1.Duration `json:"scrapeInterval,omitempty"`
	// Replicas defines the desired number of replicas
	Replicas int32 `json:"replicas,omitempty"`
	// Resources defines the resource requirements per replica
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// PlantDCoreSpec defines the desired state of PlantDCore
type PlantDCoreSpec struct {
	// KubeProxyConfig defines the desire state of PlantD Kube Proxy
	KubeProxyConfig DeploymentConfig `json:"kubeProxy,omitempty"`
	// StudioConfig defines the desire state of PlantD Studio
	StudioConfig DeploymentConfig `json:"studio,omitempty"`
	// PrometheusConfig defines the desire state of Prometheus
	PrometheusConfig PrometheusConfig `json:"prometheus,omitempty"`
	// RedisConfig defines the desire state of Redis
	RedisConfig DeploymentConfig `json:"redis,omitempty"`
}

// PlantDCoreStatus defines the observed state of PlantDCore
type PlantDCoreStatus struct {
	// KubeProxyReady shows if the PlantD Kube Proxy is ready
	KubeProxyReady bool `json:"kubeProxyReady,omitempty"`
	// StudioReady shows if the PlantD Studio is ready
	StudioReady bool `json:"studioReady,omitempty"`
	// PrometheusReady shows if the Prometheus is ready
	PrometheusReady bool `json:"prometheusReady,omitempty"`
	// RedisReady shows if the Redis is ready
	RedisReady bool `json:"redisReady,omitempty"`
	// KubeProxyStatus shows the status of the PlantD Proxy
	KubeProxyStatus string `json:"kubeProxyStatus,omitempty"`
	// StudioStatus shows the status of the PlantD Studio
	StudioStatus string `json:"studioStatus,omitempty"`
	// PrometheusStatus shows the status of the Prometheus
	PrometheusStatus string `json:"prometheusStatus,omitempty"`
	// RedisStatus shows the status of the Redis
	RedisStatus string `json:"redisStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="KubeProxyStatus",type="string",JSONPath=".status.kubeProxyStatus"
//+kubebuilder:printcolumn:name="StudioStatus",type="string",JSONPath=".status.studioStatus"
//+kubebuilder:printcolumn:name="PrometheusStatus",type="string",JSONPath=".status.prometheusStatus"
//+kubebuilder:printcolumn:name="RedisStatus",type="string",JSONPath=".status.redisStatus"

// PlantDCore is the Schema for the plantdcores API
type PlantDCore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the specifications of the PlantDCore.
	Spec PlantDCoreSpec `json:"spec,omitempty"`
	// Status defines the status of the PlantDCore.
	Status PlantDCoreStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PlantDCoreList contains a list of PlantDCore
type PlantDCoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items defines a list of PlantDCores.
	Items []PlantDCore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlantDCore{}, &PlantDCoreList{})
}
