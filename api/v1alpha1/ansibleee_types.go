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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AnsibleEESpec defines the desired state of AnsibleEE
type AnsibleEESpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Play is the playbook contents that ansible will run on execution.
	// If both Play and Roles are specified, Play takes precedence
	Play string `json:"play,omitempty"`
	// Playbook is the playbook that ansible will run on this execution
	Playbook string `json:"playbook,omitempty"`
	// Image is the container image that will execute the ansible command
	// +kubebuilder:default:="quay.io/tripleomastercentos9/openstack-tripleo-ansible-ee:current-tripleo"
	Image string `json:"image,omitempty"`
	// Args are the command plus the playbook executed by the image. If args is passed, Playbook is ignored.
	Args []string `json:"args,omitempty"`
	// Name is the name of the internal container inside the pod
	// +kubebuilder:default:="ansibleee"
	Name string `json:"name,omitempty"`
	// Env is a list containing the environment variables to pass to the pod
	Env []corev1.EnvVar `json:"env,omitempty"`
	// RestartPolicy is the policy applied to the Job on whether it needs to restart the Pod. It can be "OnFailure" or "Never".
	// +kubebuilder:default:="Never"
	RestartPolicy string `json:"restartPolicy,omitempty"`
	// UID is the userid that will be used to run the container.
	// +kubebuilder:default:=1001
	UID int64 `json:"uid,omitempty"`
	// Inventory is the inventory that the ansible playbook will use to launch the job.
	Inventory string `json:"inventory,omitempty"`
	// +kubebuilder:validation:Optional
	// ExtraMounts containing conf files and credentials
	ExtraMounts []storage.VolMounts `json:"extraMounts"`
	// Plugin is the location for the source plugin that will build dynamic inventory.
	Plugin string `json:"plugin,omitempty"`
	// BackoffLimimt allows to define the maximum number of retried executions.
	// +kubebuilder:default:=6
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`
	// TTLSecondsAfterFinished specified the number of seconds the job will be kept in Kubernetes after completion.
	// +kubebuilder:default:=86400
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`
	// Role is the description of an Ansible Role
	// If both Play and Role are specified, Play takes precedence
	Role Role `json:"roles,omitempty"`
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

// TODO:
// PluginConfig is a list where we will mount certain config settings for dynamic inventory sources.
type PluginConfig struct {
	// Name is the name of the object that we want to add to env settings
	Name string `json:"name"`

	
}

// Role describes the format of an ansible playbook destinated to run roles
type Role struct {
	// +kubebuilder:default:="Run Standalone Role"
	Name string `json:"name,omitempty"`
	// +kubebuilder:default:="{{ primary_role_name | default([]) }}:{{ deploy_target_host | default('overcloud') }}"
	Hosts string `json:"hosts,omitempty"`
	// +kubebuilder:default:=tripleo_free
	Strategy string `json:"strategy,omitempty"`
	// +kubebuilder:default:=true
	AnyErrorsFatal bool `json:"any_errors_fatal,omitempty" yaml:"any_errors_fatal,omitempty"`
	// +kubebuilder:default:=true
	Become bool `json:"become,omitempty"`
	Tasks []Task `json:"tasks"`
}

// Task describes a task centered exclusively in running import_role
type Task struct {
	Name string `json:"name"`
	ImportRole ImportRole `json:"import_role" yaml:"import_role"`
	Tags []string `json:"tags,omitempty"`
}

// ImportRole contains the actual rolename and tasks file name to execute
type ImportRole struct {
	Name string `json:"name"`
	TasksFrom string `json:"tasks_from" yaml:"tasks_from"`
}

func init() {
	SchemeBuilder.Register(&AnsibleEE{}, &AnsibleEEList{})
}
