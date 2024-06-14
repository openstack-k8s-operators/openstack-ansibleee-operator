package functional_test

import (
	"fmt"

	. "github.com/onsi/gomega" //revive:disable:dot-imports
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("OpenStackAnsibleEE Webhook", func() {

	When("User creates a valid ansibleee resource", func() {

		It("should be accepted", func() {
			DeferCleanup(th.DeleteInstance, CreateAnsibleee(ansibleeeName))

		})
	})
	When("User creates ansibleee resource referencing valid playbook", func() {
		It("should be accepted", func() {
			DeferCleanup(th.DeleteInstance, CreateAnsibleeeWithParams(
				ansibleeeName,
				"some.valid.fqcn",
				"test-image",
				"",
				"",
				map[string]interface{}{},
				[]map[string]interface{}{}))
		})
	})
	When("User creates ansibleee resource with a valid inline playbook", func() {
		It("should be accepted", func() {
			DeferCleanup(th.DeleteInstance, CreateAnsibleeeWithParams(
				ansibleeeName,
				"",
				"test-image",
				playbookContents,
				"",
				map[string]interface{}{},
				[]map[string]interface{}{},
			))
		})
	})
	When("User creates ansibleee resource with invalid playbook reference", func() {
		It("is rejected", func() {
			Eventually(
				func(_ Gomega) string {
					newInstance := map[string]interface{}{
						"apiVersion": "ansibleee.openstack.org/v1beta1",
						"kind":       "OpenStackAnsibleEE",
						"metadata": map[string]interface{}{
							"name":      ansibleeeName.Name,
							"namespace": ansibleeeName.Namespace,
						},
						"spec": map[string]interface{}{
							// this can be removed as soon as webhook is enabled in the
							// test env
							"image":    "test-image",
							"playbook": "/",
							"play":     "",
							"cmdline":  "",
						},
					}
					unstructuredObj := &unstructured.Unstructured{Object: newInstance}
					_, err := controllerutil.CreateOrPatch(th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
					return fmt.Sprintf("%s", err)
				}).Should(ContainSubstring("error checking sanity of playbook name/path"))

		})
	})
	When("User creates ansibleee resource with invalid inline playbook", func() {
		It("is rejected", func() {
			Eventually(
				func(_ Gomega) string {
					newInstance := map[string]interface{}{
						"apiVersion": "ansibleee.openstack.org/v1beta1",
						"kind":       "OpenStackAnsibleEE",
						"metadata": map[string]interface{}{
							"name":      ansibleeeName.Name,
							"namespace": ansibleeeName.Namespace,
						},
						"spec": map[string]interface{}{
							// this can be removed as soon as webhook is enabled in the
							// test env
							"image":            "test-image",
							"playbook":         "",
							"playbookContents": malformedPlaybook,
							"cmdline":          "",
						},
					}
					unstructuredObj := &unstructured.Unstructured{Object: newInstance}
					_, err := controllerutil.CreateOrPatch(th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
					return fmt.Sprintf("%s", err)
				}).Should(ContainSubstring("error checking sanity of an inline play"))
		})
	})
})
