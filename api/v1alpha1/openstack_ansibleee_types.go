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
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OpenStackAnsibleEESpec defines the desired state of OpenStackAnsibleEE
type OpenStackAnsibleEESpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Play is the playbook contents that ansible will run on execution.
	// If both Play and Roles are specified, Play takes precedence
	Play string `json:"play,omitempty"`
	// Playbook is the playbook that ansible will run on this execution
	Playbook string `json:"playbook,omitempty"`
	// Image is the container image that will execute the ansible command
	// +kubebuilder:default:="quay.io/openstack-k8s-operators/openstack-ansibleee-runner:latest"
	Image string `json:"image,omitempty"`
	// Args are the command plus the playbook executed by the image. If args is passed, Playbook is ignored.
	Args []string `json:"args,omitempty"`
	// Name is the name of the internal container inside the pod
	// +kubebuilder:default:="openstackansibleee"
	Name string `json:"name,omitempty"`
	// Env is a list containing the environment variables to pass to the pod
	Env []corev1.EnvVar `json:"env,omitempty"`
	// RestartPolicy is the policy applied to the Job on whether it needs to restart the Pod. It can be "OnFailure" or "Never".
	// RestartPolicy default: Never
	// +kubebuilder:validation:Enum:=OnFailure;Never
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:OnFailure","urn:alm:descriptor:com.tectonic.ui:select:Never"}
	// +kubebuilder:default:="Never"
	RestartPolicy string `json:"restartPolicy,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// PreserveJobs - do not delete jobs after they finished e.g. to check logs
	PreserveJobs bool `json:"preserveJobs"`
	// UID is the userid that will be used to run the container.
	// +kubebuilder:default:=1001
	UID int64 `json:"uid,omitempty"`
	// Inventory is the inventory that the ansible playbook will use to launch the job.
	Inventory string `json:"inventory,omitempty"`
	// +kubebuilder:validation:Optional
	// ExtraMounts containing conf files and credentials
	ExtraMounts []storage.VolMounts `json:"extraMounts,omitempty"`
	// BackoffLimimt allows to define the maximum number of retried executions (defaults to 6).
	// +kubebuilder:default:=6
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`
	// Role is the description of an Ansible Role
	// If both Play and Role are specified, Play takes precedence
	// +kubebuilder:validation:Optional
	Role *Role `json:"roles,omitempty"`
	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to expose the services to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`
	// +kubebuilder:validation:Optional
	// CmdLine is the command line passed to ansible-runner
	CmdLine string `json:"cmdLine,omitempty"`
	// +kubebuilder:validation:Optional
	// InitContainers allows the passing of an array of containers that will be executed before the ansibleee execution itself
	InitContainers []corev1.Container `json:"initContainers,omitempty"`
	// DeployIdentifier is a generated UUID set as input on OpenStackAnsibleEE resources
	// so that the OpenStackAnsibleEE controller can determine job input uniqueness.
	// It is generated on each new deploy request (when DeployStrategy.Deploy is changed to true).
	// +kubebuilder:validation:Optional
	DeployIdentifier string `json:"deployIdentifier"`
}

// OpenStackAnsibleEEStatus defines the observed state of OpenStackAnsibleEE
type OpenStackAnsibleEEStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// NetworkAttachments status of the deployment pods
	NetworkAttachments map[string][]string `json:"networkAttachments,omitempty"`

	// +kubebuilder:validation:Enum:=Pending;Running;Succeeded;Failed
	// +kubebuilder:default:=Pending
	// JobStatus status of the executed job (Pending/Running/Succeeded/Failed)
	JobStatus string `json:"JobStatus,omitempty" optional:"true"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+operator-sdk:csv:customresourcedefinitions:displayName="OpenStack Ansible EE"
// +kubebuilder:resource:shortName=osaee;osaees;osansible;osansibles
//+kubebuilder:printcolumn:name="NetworkAttachments",type="string",JSONPath=".spec.networkAttachments",description="NetworkAttachments"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

// OpenStackAnsibleEE is the Schema for the openstackansibleees API
type OpenStackAnsibleEE struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenStackAnsibleEESpec   `json:"spec,omitempty"`
	Status OpenStackAnsibleEEStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenStackAnsibleEEList contains a list of OpenStackAnsibleEE
type OpenStackAnsibleEEList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenStackAnsibleEE `json:"items"`
}

// Config is a specification of where to mount a certain ConfigMap object
type Config struct {
	// Name is the name of the ConfigMap that we want to mount
	Name string `json:"name"`
	// MountPoint is the directory of the container where the ConfigMap will be mounted
	MountPath string `json:"mountpath"`
}

// Role describes the format of an ansible playbook destinated to run roles
type Role struct {
	// +kubebuilder:default:="Run Standalone Role"
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// +kubebuilder:default:="{{ primary_role_name | default([]) }}:{{ deploy_target_host | default('overcloud') }}"
	Hosts string `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	// +kubebuilder:default:=linear
	// strategy defaults to linear
	Strategy string `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// any_errors_fatal defaults to true
	AnyErrorsFatal bool `json:"any_errors_fatal,omitempty" yaml:"any_errors_fatal,omitempty"`
	// +kubebuilder:default:=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// become defaults to false
	Become bool `json:"become,omitempty" yaml:"become,omitempty"`
	// +kubebuilder:default:=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// gather_facts defaults to false
	GatherFacts bool   `json:"gather_facts,omitempty" yaml:"gather_facts,omitempty"`
	Tasks       []Task `json:"tasks,omitempty" yaml:"tasks,omitempty"`
}

// Task describes a task centered exclusively in running import_role
type Task struct {
	Name       string     `json:"name" yaml:"name"`
	ImportRole ImportRole `json:"import_role" yaml:"import_role"`
	Vars       []string   `json:"vars,omitempty" yaml:"vars,omitempty"`
	When       string     `json:"when,omitempty" yaml:"when,omitempty"`
	Tags       []string   `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// ImportRole contains the actual rolename and tasks file name to execute
type ImportRole struct {
	Name      string `json:"name" yaml:"name"`
	TasksFrom string `json:"tasks_from,omitempty" yaml:"tasks_from,omitempty"`
}

func init() {
	SchemeBuilder.Register(&OpenStackAnsibleEE{}, &OpenStackAnsibleEEList{})
}

// IsReady - returns true if the OpenStackAnsibleEE is ready
func (instance OpenStackAnsibleEE) IsReady() bool {
	return instance.Status.Conditions.IsTrue(AnsibleExecutionJobReadyCondition)
}
