/*
Copyright 2022.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AnsibleEESpec defines the desired state of AnsibleEE
type AnsibleEESpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Playbook is the playbook that ansible will run on this execution
	Playbook string `json:"playbook,omitempty"`
	// Image is the container image that will execute the ansible command
	// +kubebuilder:default:="quay.io/jlarriba/openstack-tripleo-ansible-ee"
	Image string `json:"image,omitempty"`
	// Args are the command plus the playbook executed by the image. If args is passed, Playbook is ignored.
	Args []string `json:"args,omitempty"`
	// Name is the name of the internal container inside the pod
	// +kubebuilder:default:="ansibleee"
	Name string `json:"name,omitempty"`
	// RestartPolicy is the policy applied to the Job on whether it needs to restart the Pod. It can be "OnFailure" or "Never".
	// +kubebuilder:default:="Never"
	RestartPolicy string `json:"restartPolicy,omitempty"`
	// Uid is the userid that will be used to run the container.
	// +kubebuilder:default:=1001
	Uid int64 `json:"uid,omitempty"`
	// Inventory is the inventory that the ansible playbook will use to launch the job.
	Inventory string `json:"inventory,omitempty"`
	// Config allows to pass a list of Config
	Configs []Config `json:"configs,omitempty"`
	// BackoffLimimt allows to define the maximum number of retried executions.
	// +kubebuilder:default:=6
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`
	// TTLSecondsAfterFinished specified the number of seconds the job will be kept in Kubernetes after completion.
	// +kubebuilder:default:=86400
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`
}

// AnsibleEEStatus defines the observed state of AnsibleEE
type AnsibleEEStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Nodes are the names of the ansibleee pods
	Nodes []string `json:"nodes"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AnsibleEE is the Schema for the ansibleees API
type AnsibleEE struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AnsibleEESpec   `json:"spec,omitempty"`
	Status AnsibleEEStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AnsibleEEList contains a list of AnsibleEE
type AnsibleEEList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AnsibleEE `json:"items"`
}

// Config is a specification of where to mount a certain ConfigMap object
type Config struct {
	// Name is the name of the ConfigMap that we want to mount
	Name string `json:"name"`
	// MountPoint is the directory of the container where the ConfigMap will be mounted
	MountPath string `json:"mountpath"`
}

func init() {
	SchemeBuilder.Register(&AnsibleEE{}, &AnsibleEEList{})
}
