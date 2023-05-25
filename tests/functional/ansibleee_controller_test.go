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

package functional_test

import (
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	. "github.com/openstack-k8s-operators/lib-common/modules/test/helpers"
	"github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

var _ = Describe("Ansibleee controller", func() {
	When("Ansibleee CR instance is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateAnsibleee(ansibleeeName))
		})

		It("runs a Job", func() {
			th.ExpectConditionWithDetails(
				ansibleeeName,
				ConditionGetterFunc(AnsibleeeConditionGetter),
				v1alpha1.AnsibleExecutionJobReadyCondition,
				corev1.ConditionFalse,
				condition.RequestedReason,
				"AnsibleExecutionJob is running",
			)
			th.ExpectCondition(
				ansibleeeName,
				ConditionGetterFunc(AnsibleeeConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionUnknown,
			)
		})
	})
})
