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
	"time"

	yaml "gopkg.in/yaml.v3"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"context"

	"github.com/go-logr/logr"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	helper "github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	util "github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	redhatcomv1alpha1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

// OpenStackAnsibleEEReconciler reconciles a OpenStackAnsibleEE object
type OpenStackAnsibleEEReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Log     logr.Logger
	Scheme  *runtime.Scheme
}

// +kubebuilder:rbac:groups=ansibleee.openstack.org,resources=openstackansibleees,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ansibleee.openstack.org,resources=openstackansibleees/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ansibleee.openstack.org,resources=openstackansibleees/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OpenStackAnsibleEE object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *OpenStackAnsibleEEReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	instance, helper, err := r.getOpenStackAnsibleeeInstance(ctx, req)
	if err != nil || instance.Name == "" {
		return ctrl.Result{}, err
	}

	// networks to attach to
	for _, netAtt := range instance.Spec.NetworkAttachments {
		_, err := nad.GetNADWithName(ctx, helper, netAtt, instance.Namespace)
		if err != nil {
			if errors.IsNotFound(err) {
				instance.Status.Conditions.Set(condition.FalseCondition(
					condition.NetworkAttachmentsReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					condition.NetworkAttachmentsReadyWaitingMessage,
					netAtt))
				return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("network-attachment-definition %s not found", netAtt)
			}
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NetworkAttachmentsReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NetworkAttachmentsReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
	}

	serviceAnnotations, err := nad.CreateNetworksAnnotation(instance.Namespace, instance.Spec.NetworkAttachments)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed create network annotation from %s: %w",
			instance.Spec.NetworkAttachments, err)
	}

	// Check if the job already exists, if not create a new one
	foundJob := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, foundJob)
	if err != nil && errors.IsNotFound(err) {
		// Define a new job
		job, err := r.jobForOpenStackAnsibleEE(instance, helper, serviceAnnotations)
		if err != nil {
			return ctrl.Result{}, err
		}
		util.LogForObject(
			helper,
			fmt.Sprintf("Creating a new Job: Job.Namespace %s Job.Name %s\n", job.Namespace, job.Name),
			instance,
		)
		err = r.Create(ctx, job)
		if err != nil {
			util.LogErrorForObject(helper, err, err.Error(), instance)
			return ctrl.Result{}, err
		}
		util.LogForObject(
			helper,
			fmt.Sprintf("Job: Job.Namespace %s Job.Name %s created successfully - return and requeue\n", job.Namespace, job.Name),
			instance,
		)
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		util.LogErrorForObject(helper, err, err.Error(), instance)
		return ctrl.Result{}, err
	}
	if instance.Status.NetworkAttachments == nil {
		instance.Status.NetworkAttachments = map[string][]string{}
	}

	return ctrl.Result{}, nil
}

func (r *OpenStackAnsibleEEReconciler) getOpenStackAnsibleeeInstance(ctx context.Context, req ctrl.Request) (*redhatcomv1alpha1.OpenStackAnsibleEE, *helper.Helper, error) {
	// Fetch the OpenStackAnsibleEE instance
	instance := &redhatcomv1alpha1.OpenStackAnsibleEE{}
	helper, err := helper.NewHelper(
		instance,
		r.Client,
		r.Kclient,
		r.Scheme,
		r.Log,
	)

	if err != nil {
		// helper might be nil, so can't use util.LogErrorForObject since it requires helper as first arg
		r.Log.Error(err, fmt.Sprintf("Unable to acquire helper for  OpenStackAnsibleEE %s", instance.Name))
		return instance, nil, err
	}

	err = r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			util.LogForObject(
				helper,
				"OpenStackAnsibleEE resource not found. Ignoring since object must be deleted",
				instance,
			)
			return &redhatcomv1alpha1.OpenStackAnsibleEE{}, helper, nil
		}
		// Error reading the object - requeue the request.
		util.LogErrorForObject(helper, err, err.Error(), instance)
		return instance, helper, err
	}

	return instance, helper, nil
}

// jobForOpenStackAnsibleEE returns a openstackansibleee Job object
func (r *OpenStackAnsibleEEReconciler) jobForOpenStackAnsibleEE(instance *redhatcomv1alpha1.OpenStackAnsibleEE, h *helper.Helper, annotations map[string]string) (*batchv1.Job, error) {
	ls := labelsForOpenStackAnsibleEE(instance.Name)

	args := instance.Spec.Args

	if len(args) == 0 {
		if len(instance.Spec.Playbook) == 0 {
			instance.Spec.Playbook = "playbook.yaml"
		}
		args = []string{"ansible-runner", "run", "/runner", "-p", instance.Spec.Playbook}
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: instance.Spec.TTLSecondsAfterFinished,
			BackoffLimit:            instance.Spec.BackoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
					Labels:      ls,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicy(instance.Spec.RestartPolicy),
					Containers: []corev1.Container{{
						ImagePullPolicy: "Always",
						Image:           instance.Spec.Image,
						Name:            instance.Spec.Name,
						Args:            args,
					}},
				},
			},
		},
	}

	if len(instance.Spec.Inventory) > 0 {
		addInventory(instance, job)
	}
	if len(instance.Spec.Play) > 0 {
		addPlay(instance, job)
	} else {
		addRoles(instance, h, job)
	}
	if len(instance.Spec.CmdLine) > 0 {
		addCmdLine(instance, job)
	}
	addMounts(instance, job)

	// Set OpenStackAnsibleEE instance as the owner and controller
	err := ctrl.SetControllerReference(instance, job, r.Scheme)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// labelsForOpenStackAnsibleEE returns the labels for selecting the resources
// belonging to the given openstackansibleee CR name.
func labelsForOpenStackAnsibleEE(name string) map[string]string {
	return map[string]string{"app": "openstackansibleee", "openstackansibleee_cr": name}
}

func addMounts(instance *redhatcomv1alpha1.OpenStackAnsibleEE, job *batchv1.Job) {
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume

	// ExtraMounts propagation: for each volume defined in the top-level CR
	// the propagation function provided by lib-common/modules/storage is
	// called, and the resulting corev1.Volumes and corev1.Mounts are added
	// to the main list defined by the ansible operator
	for _, exv := range instance.Spec.ExtraMounts {
		for _, vol := range exv.Propagate([]storage.PropagationType{storage.Compute}) {
			volumes = append(volumes, vol.Volumes...)
			volumeMounts = append(volumeMounts, vol.Mounts...)
		}
	}

	job.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
	job.Spec.Template.Spec.Volumes = volumes
}

func addRoles(instance *redhatcomv1alpha1.OpenStackAnsibleEE, h *helper.Helper, job *batchv1.Job) {
	var roles []*redhatcomv1alpha1.Role
	roles = append(roles, &instance.Spec.Role)
	d, err := yaml.Marshal(&roles)
	if err != nil {
		util.LogErrorForObject(h, err, err.Error(), instance)
	}

	var roleEnvVar corev1.EnvVar
	roleEnvVar.Name = "RUNNER_PLAYBOOK"
	roleEnvVar.Value = "\n" + string(d) + "\n\n"
	instance.Spec.Env = append(instance.Spec.Env, roleEnvVar)
	job.Spec.Template.Spec.Containers[0].Env = instance.Spec.Env
}

func addPlay(instance *redhatcomv1alpha1.OpenStackAnsibleEE, job *batchv1.Job) {
	var playEnvVar corev1.EnvVar
	playEnvVar.Name = "RUNNER_PLAYBOOK"
	playEnvVar.Value = "\n" + instance.Spec.Play + "\n\n"
	instance.Spec.Env = append(instance.Spec.Env, playEnvVar)
	job.Spec.Template.Spec.Containers[0].Env = instance.Spec.Env
}

func addInventory(instance *redhatcomv1alpha1.OpenStackAnsibleEE, job *batchv1.Job) {
	var invEnvVar corev1.EnvVar
	invEnvVar.Name = "RUNNER_INVENTORY"
	invEnvVar.Value = "\n" + instance.Spec.Inventory + "\n\n"
	instance.Spec.Env = append(instance.Spec.Env, invEnvVar)
	job.Spec.Template.Spec.Containers[0].Env = instance.Spec.Env
}

func addCmdLine(instance *redhatcomv1alpha1.OpenStackAnsibleEE, job *batchv1.Job) {
	var cmdLineEnvVar corev1.EnvVar
	cmdLineEnvVar.Name = "RUNNER_CMDLINE"
	cmdLineEnvVar.Value = "\n" + instance.Spec.CmdLine + "\n\n"
	instance.Spec.Env = append(instance.Spec.Env, cmdLineEnvVar)
	job.Spec.Template.Spec.Containers[0].Env = instance.Spec.Env
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackAnsibleEEReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&redhatcomv1alpha1.OpenStackAnsibleEE{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
