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

import condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"

// AnsibleEE Condition Types.
const (
	// AnsibleExecutionJobReadyCondition Status=True condition indicates
	// AnsibleExecutionJob is ready.
	AnsibleExecutionJobReadyCondition condition.Type = "AnsibleExecutionJobReady"
)

// Common Messages used by AnsibleEE objects.
const (
	//
	// AnsibleExecutionJob condition messages
	//
	// AnsibleExecutionJobInitMessage
	AnsibleExecutionJobInitMessage = "AnsibleExecutionJob not started"

	// AnsibleExecutionJobReadyMessage
	AnsibleExecutionJobReadyMessage = "AnsibleExecutionJob ready"

	// AnsibleExecutionJobNotFoundMessage
	AnsibleExecutionJobNotFoundMessage = "AnsibleExecutionJob not found"

	// AnsibleExecutionJobWaitingMessage
	AnsibleExecutionJobWaitingMessage = "AnsibleExecutionJob not yet ready"

	// AnsibleExecutionJobErrorMessage
	AnsibleExecutionJobErrorMessage = "AnsibleExecutionJob error occured %s"
)
