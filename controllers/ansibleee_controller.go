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

package controllers

import (
	"fmt"
	"reflect"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redhatcomv1alpha1 "github.com/jlarriba/ansibleee-operator/api/v1alpha1"
)

// AnsibleEEReconciler reconciles a AnsibleEE object
type AnsibleEEReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=redhat.com,resources=ansibleees,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=redhat.com,resources=ansibleees/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=redhat.com,resources=ansibleees/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AnsibleEE object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *AnsibleEEReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//log := r.Log.WithValues("ansibleee", req.NamespacedName)

	// Fetch the AnsibleEE instance
	ansibleee := &redhatcomv1alpha1.AnsibleEE{}
	err := r.Get(ctx, req.NamespacedName, ansibleee)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			fmt.Println("AnsibleEE resource not found. Ignoring since object must be deleted")
			//log.Info("AnsibleEE resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		fmt.Println(err.Error())
		//log.Error(err, "Failed to get AnsibleEE")
		return ctrl.Result{}, err
	}

	// Check if the job already exists, if not create a new one
	found := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: ansibleee.Name, Namespace: ansibleee.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new job
		job := r.jobForAnsibleEE(ansibleee)
		fmt.Printf("Creating a new Job: Job.Namespace %s Job.Name %s\n", job.Namespace, job.Name)
		err = r.Create(ctx, job)
		if err != nil {
			fmt.Println(err.Error())
			return ctrl.Result{}, err
		}
		fmt.Println("job created successfully - return and requeue")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		fmt.Println(err.Error())
		//log.Error(err, "Failed to get Job")
		return ctrl.Result{}, err
	}

	// Update the AnsibleEE status with the pod names
	// List the pods for this ansibleee's job
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(ansibleee.Namespace),
		client.MatchingLabels(labelsForAnsibleEE(ansibleee.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		//fmt.Println(err.Error())
		//log.Error(err, "Failed to list pods", "AnsibleEE.Namespace", ansibleee.Namespace, "AnsibleEE.Name", ansibleee.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, ansibleee.Status.Nodes) {
		ansibleee.Status.Nodes = podNames
		err := r.Status().Update(ctx, ansibleee)
		if err != nil {
			//fmt.Println(err.Error())
			//log.Error(err, "Failed to update AnsibleEE status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// jobForAnsibleEE returns a ansibleee Job object
func (r *AnsibleEEReconciler) jobForAnsibleEE(m *redhatcomv1alpha1.AnsibleEE) *batchv1.Job {
	ls := labelsForAnsibleEE(m.Name)

	args := m.Spec.Args

	if len(args) == 0 {
		args = []string{"ansible-runner run /runner -p", m.Spec.Playbook}
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicy(m.Spec.RestartPolicy),
					Containers: []corev1.Container{{
						Image: m.Spec.Image,
						Name:  m.Spec.Name,
						Args:  args,
					}},
				},
			},
		},
	}

	// Set AnsibleEE instance as the owner and controller
	ctrl.SetControllerReference(m, job, r.Scheme)
	return job
}

// labelsForAnsibleEE returns the labels for selecting the resources
// belonging to the given ansibleee CR name.
func labelsForAnsibleEE(name string) map[string]string {
	return map[string]string{"app": "ansibleee", "ansibleee_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// SetupWithManager sets up the controller with the Manager.
func (r *AnsibleEEReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&redhatcomv1alpha1.AnsibleEE{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
