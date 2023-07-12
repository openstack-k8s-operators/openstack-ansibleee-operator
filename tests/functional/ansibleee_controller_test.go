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
	"fmt"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	"github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

var _ = Describe("Ansibleee controller", func() {
	When("Ansibleee CR instance is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateAnsibleee(ansibleeeName))
		})

		It("runs a job and reports when it succeeds", func() {
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
				corev1.ConditionFalse,
			)
			ansibleee := GetAnsibleee(ansibleeeName)
			Expect(ansibleee.Status.JobStatus).To(Equal("Running"))
			Expect(ansibleee.Status.Hash).To(HaveKey("input"))
			Expect(ansibleee.Status.Hash).NotTo(HaveKey("ansibleee"))

			// simulate that the job succeeds
			th.SimulateJobSuccess(ansibleeeName)

			th.ExpectCondition(
				ansibleeeName,
				ConditionGetterFunc(AnsibleeeConditionGetter),
				v1alpha1.AnsibleExecutionJobReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				ansibleeeName,
				ConditionGetterFunc(AnsibleeeConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionTrue,
			)
			ansibleee = GetAnsibleee(ansibleeeName)
			Expect(ansibleee.Status.JobStatus).To(Equal("Succeeded"))
			Expect(ansibleee.Status.Hash).To(HaveKey("input"))
			Expect(ansibleee.Status.Hash).To(HaveKey("ansibleee"))
		})

		It("runs a job and reports if it fails", func() {
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
				corev1.ConditionFalse,
			)
			ansibleee := GetAnsibleee(ansibleeeName)
			Expect(ansibleee.Status.JobStatus).To(Equal("Running"))
			Expect(ansibleee.Status.Hash).To(HaveKey("input"))
			Expect(ansibleee.Status.Hash).NotTo(HaveKey("ansibleee"))

			// simulate that the job fails
			th.SimulateJobFailure(ansibleeeName)

			th.ExpectConditionWithDetails(
				ansibleeeName,
				ConditionGetterFunc(AnsibleeeConditionGetter),
				v1alpha1.AnsibleExecutionJobReadyCondition,
				corev1.ConditionFalse,
				condition.ErrorReason,
				fmt.Sprintf(
					"AnsibleExecutionJob error occured job.name %s job.namespace %s failed",
					ansibleee.Name, ansibleeeName.Namespace,
				),
			)
			th.ExpectCondition(
				ansibleeeName,
				ConditionGetterFunc(AnsibleeeConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
			ansibleee = GetAnsibleee(ansibleeeName)
			Expect(ansibleee.Status.JobStatus).To(Equal("Failed"))
			Expect(ansibleee.Status.Hash).To(HaveKey("input"))
			Expect(ansibleee.Status.Hash).NotTo(HaveKey("ansibleee"))
		})

		It("re-runs the job if the input of the job changes", func() {
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
				corev1.ConditionFalse,
			)
			// simulate that the job succeeds
			th.SimulateJobSuccess(ansibleeeName)

			th.ExpectCondition(
				ansibleeeName,
				ConditionGetterFunc(AnsibleeeConditionGetter),
				v1alpha1.AnsibleExecutionJobReadyCondition,
				corev1.ConditionTrue,
			)
			ansibleee := GetAnsibleee(ansibleeeName)
			Expect(ansibleee.Status.JobStatus).To(Equal("Succeeded"))
			Expect(ansibleee.Status.Hash).To(HaveKey("input"))
			originalInputHash := ansibleee.Status.Hash["input"]
			Expect(ansibleee.Status.Hash).To(HaveKey("ansibleee"))
			originalJobHash := ansibleee.Status.Hash["ansibleee"]

			// change some input to the Ansibleee CR
			Eventually(func(g Gomega) {
				ansibleee := GetAnsibleee(ansibleeeName)
				ansibleee.Spec.Args = []string{"--debug"}

				g.Expect(k8sClient.Update(ctx, ansibleee)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			logger.Info("Updated Ansibleee CR")

			// A new job should be started
			th.ExpectConditionWithDetails(
				ansibleeeName,
				ConditionGetterFunc(AnsibleeeConditionGetter),
				v1alpha1.AnsibleExecutionJobReadyCondition,
				corev1.ConditionFalse,
				condition.RequestedReason,
				"AnsibleExecutionJob is running",
			)
			ansibleee = GetAnsibleee(ansibleeeName)
			Expect(ansibleee.Status.JobStatus).To(Equal("Running"))

			// simulate that the second job succeeds
			th.SimulateJobSuccess(ansibleeeName)

			th.ExpectCondition(
				ansibleeeName,
				ConditionGetterFunc(AnsibleeeConditionGetter),
				v1alpha1.AnsibleExecutionJobReadyCondition,
				corev1.ConditionTrue,
			)
			ansibleee = GetAnsibleee(ansibleeeName)
			Expect(ansibleee.Status.JobStatus).To(Equal("Succeeded"))
			Expect(ansibleee.Status.Hash["input"]).NotTo(Equal(originalInputHash))
			Expect(ansibleee.Status.Hash["ansibleee"]).NotTo(Equal(originalJobHash))
		})
	})
})
