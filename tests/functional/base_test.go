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
	. "github.com/onsi/gomega"

	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// This constant must NOT use tabs, as it as raw string passed to the ansible-runner
	play = `
- name: Print hello world
  hosts: all
  tasks:
    - name: Using debug statement
      ansible.builtin.debug:
        msg: "Hello, world this is ansibleee-play.yaml"`
)

func GetAnsibleee(name types.NamespacedName) *v1alpha1.OpenStackAnsibleEE {
	instance := &v1alpha1.OpenStackAnsibleEE{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

func AnsibleeeConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetAnsibleee(name)
	return instance.Status.Conditions
}

func CreateAnsibleee(name types.NamespacedName) client.Object {
	raw := map[string]interface{}{
		"apiVersion": "ansibleee.openstack.org/v1alpha1",
		"kind":       "OpenStackAnsibleEE",
		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": map[string]interface{}{
			// this can be removed as soon as webhook is enabled in the
			// test env
			"image": "test-image",
		},
	}
	return th.CreateUnstructured(raw)
}

func CreateAnsibleeeWithParams(
	name types.NamespacedName, playbook string, image string, play string,
	debug bool, cmdline string, extraVars map[string]interface{}) client.Object {

	raw := map[string]interface{}{
		"apiVersion": "ansibleee.openstack.org/v1alpha1",
		"kind":       "OpenStackAnsibleEE",
		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": map[string]interface{}{
			// this can be removed as soon as webhook is enabled in the
			// test env
			"image":     image,
			"playbook":  playbook,
			"play":      play,
			"debug":     debug,
			"cmdline":   cmdline,
			"extraVars": extraVars,
		},
	}

	return th.CreateUnstructured(raw)
}
