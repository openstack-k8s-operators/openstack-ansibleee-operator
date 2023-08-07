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
	job "github.com/openstack-k8s-operators/lib-common/modules/common/job"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	util "github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	redhatcomv1alpha1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1alpha1"
)

const (
	ansibleeeJobType       = "ansibleee"
	ansibleeeInputHashName = "input"
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
// the AnsibleEE object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *OpenStackAnsibleEEReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {

	instance, err := r.getOpenStackAnsibleeeInstance(ctx, req)
	if err != nil || instance.Name == "" {
		return ctrl.Result{}, err
	}

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
		return ctrl.Result{}, err
	}

	// Always patch the instance status when exiting this function so we can
	// persist any changes.
	defer func() {
		// update the overall status condition if service is ready
		if instance.IsReady() {
			instance.Status.Conditions.MarkTrue(condition.ReadyCondition, redhatcomv1alpha1.AnsibleExecutionJobReadyMessage)
		} else {
			// something is not ready so reset the Ready condition
			instance.Status.Conditions.MarkUnknown(
				condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage)
			// and recalculate it based on the state of the rest of the conditions
			instance.Status.Conditions.Set(instance.Status.Conditions.Mirror(condition.ReadyCondition))
		}

		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			r.Log.Error(_err, "PatchInstance error")
			_err = err
			return
		}
	}()

	// Initialize Status
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = condition.Conditions{}

		cl := condition.CreateList(
			condition.UnknownCondition(redhatcomv1alpha1.AnsibleExecutionJobReadyCondition, condition.InitReason, redhatcomv1alpha1.AnsibleExecutionJobInitMessage),
		)

		instance.Status.Conditions.Init(&cl)

		// Register overall status immediately to have an early feedback e.g.
		// in the cli
		return ctrl.Result{}, nil
	}

	// Initialize Status fields
	if instance.Status.Hash == nil {
		instance.Status.Hash = map[string]string{}
	}
	if instance.Status.NetworkAttachments == nil {
		instance.Status.NetworkAttachments = map[string][]string{}
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

	currentJobHash := instance.Status.Hash[ansibleeeJobType]

	// Define a new job
	jobDef, err := r.jobForOpenStackAnsibleEE(instance, helper, serviceAnnotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	configMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: instance.Spec.EnvConfigMapName, Namespace: instance.Namespace}, configMap)
	if err != nil && !errors.IsNotFound(err) {
		r.Log.Error(err, err.Error())
		return ctrl.Result{}, err
	} else if err == nil {
		addEnvFrom(instance, jobDef)
	}

	ansibleeeJob := job.NewJob(
		jobDef,
		ansibleeeJobType,
		instance.Spec.PreserveJobs,
		time.Duration(5)*time.Second,
		currentJobHash,
	)

	ctrlResult, err := ansibleeeJob.DoJob(
		ctx,
		helper,
	)

	if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			redhatcomv1alpha1.AnsibleExecutionJobReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			redhatcomv1alpha1.AnsibleExecutionJobWaitingMessage))
		instance.Status.JobStatus = redhatcomv1alpha1.JobStatusRunning
		return ctrlResult, nil
	}

	if err != nil {
		err = fmt.Errorf("job.name %s job.namespace %s failed", jobDef.Name, jobDef.Namespace)
		instance.Status.Conditions.Set(condition.FalseCondition(
			redhatcomv1alpha1.AnsibleExecutionJobReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			redhatcomv1alpha1.AnsibleExecutionJobErrorMessage,
			err.Error()))
		instance.Status.JobStatus = redhatcomv1alpha1.JobStatusFailed
		return ctrl.Result{}, err
	}

	if ansibleeeJob.HasChanged() {
		instance.Status.Hash[ansibleeeJobType] = ansibleeeJob.GetHash()
		r.Log.Info(fmt.Sprintf("AnsibleEE CR '%s' - Job %s hash added - %s", instance.Name, jobDef.Name, instance.Status.Hash[ansibleeeJobType]))
	}

	instance.Status.Conditions.MarkTrue(redhatcomv1alpha1.AnsibleExecutionJobReadyCondition, redhatcomv1alpha1.AnsibleExecutionJobReadyMessage)
	instance.Status.JobStatus = redhatcomv1alpha1.JobStatusSucceeded

	r.Log.Info(fmt.Sprintf("Reconciled AnsibleEE '%s' successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *OpenStackAnsibleEEReconciler) getOpenStackAnsibleeeInstance(ctx context.Context, req ctrl.Request) (*redhatcomv1alpha1.OpenStackAnsibleEE, error) {
	// Fetch the OpenStackAnsibleEE instance
	instance := &redhatcomv1alpha1.OpenStackAnsibleEE{}

	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info("OpenStackAnsibleEE resource not found. Ignoring since object must be deleted")
			return &redhatcomv1alpha1.OpenStackAnsibleEE{}, nil
		}
		// Error reading the object - requeue the request.
		r.Log.Error(err, err.Error())
		return instance, err
	}

	return instance, nil
}

// jobForOpenStackAnsibleEE returns a openstackansibleee Job object
func (r *OpenStackAnsibleEEReconciler) jobForOpenStackAnsibleEE(
	instance *redhatcomv1alpha1.OpenStackAnsibleEE,
	h *helper.Helper,
	annotations map[string]string) (*batchv1.Job, error) {
	labels := instance.GetObjectMeta().GetLabels()
	var deployIdentifier string
	if len(labels["deployIdentifier"]) > 0 {
		deployIdentifier = labels["deployIdentifier"]
	} else {
		deployIdentifier = ""
	}
	ls := labelsForOpenStackAnsibleEE(instance.Name, deployIdentifier)

	args := instance.Spec.Args

	if len(args) == 0 {
		if len(instance.Spec.Playbook) == 0 {
			instance.Spec.Playbook = "playbook.yaml"
		}
		args = []string{"ansible-runner", "run", "/runner", "-p", instance.Spec.Playbook}
	}

	hasIdentifier := false
	for _, arg := range args {
		// ansible runner identifier
		// https://ansible-runner.readthedocs.io/en/stable/intro/#artifactdir
		if arg == "-i" || arg == "--ident" {
			hasIdentifier = true
			break
		}
	}

	if !hasIdentifier {
		identifier := instance.Name
		args = append(args, []string{"-i", identifier}...)
	}

	// Override args list if we are in a debug mode
	if instance.Spec.Debug {
		args = []string{"sleep", "1d"}
		r.Log.Info(fmt.Sprintf("Instance %s will be running in debug mode.", instance.Name))
	}
	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicy(instance.Spec.RestartPolicy),
		Containers: []corev1.Container{{
			ImagePullPolicy: "Always",
			Image:           instance.Spec.Image,
			Name:            instance.Spec.Name,
			Args:            args,
			Env:             instance.Spec.Env,
		}},
	}

	if instance.Spec.DNSConfig != nil {
		podSpec.DNSConfig = instance.Spec.DNSConfig
		podSpec.DNSPolicy = "None"
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: instance.Spec.BackoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
					Labels:      ls,
				},
				Spec: podSpec,
			},
		},
	}

	//Populate hash
	hashes := make(map[string]string)

	if len(instance.Spec.InitContainers) > 0 {
		job.Spec.Template.Spec.InitContainers = instance.Spec.InitContainers
	}
	if len(instance.Spec.ServiceAccountName) > 0 {
		job.Spec.Template.Spec.ServiceAccountName = instance.Spec.ServiceAccountName
	}
	if len(instance.Spec.Inventory) > 0 {
		addInventory(instance, h, job, hashes)
	}
	if len(instance.Spec.Play) > 0 {
		addPlay(instance, h, job, hashes)
	} else if instance.Spec.Role != nil {
		addRoles(instance, h, job, hashes)
	} else if len(instance.Spec.Playbook) > 0 {
		// As we set "playbook.yaml" as default
		// we need to ensure that Play and Role are empty before addPlaybook
		addPlaybook(instance, h, job, hashes)
	}
	if len(instance.Spec.CmdLine) > 0 && !instance.Spec.Debug {
		// RUNNER_CMDLINE environment variable should only be set
		// if the operator isn't running in a debug mode.
		addCmdLine(instance, h, job, hashes)
	}
	if len(labels["deployIdentifier"]) > 0 {
		hashes["deployIdentifier"] = labels["deployIdentifier"]
	}

	addMounts(instance, job)

	inputHash, errorHash := hashOfInputHashes(hashes)
	if errorHash != nil {
		return nil, fmt.Errorf("error generating hash of input hashes: %w", errorHash)
	}
	instance.Status.Hash[ansibleeeInputHashName] = inputHash

	// Set OpenStackAnsibleEE instance as the owner and controller
	err := ctrl.SetControllerReference(instance, job, r.Scheme)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// labelsForOpenStackAnsibleEE returns the labels for selecting the resources
// belonging to the given openstackansibleee CR name.
func labelsForOpenStackAnsibleEE(name string, deployIdentifier string) map[string]string {
	return map[string]string{
		"app":                   "openstackansibleee",
		"deployIdentifier":      deployIdentifier,
		"openstackansibleee_cr": name,
	}
}

func addEnvFrom(instance *redhatcomv1alpha1.OpenStackAnsibleEE, job *batchv1.Job) {
	job.Spec.Template.Spec.Containers[0].EnvFrom = []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: instance.Spec.EnvConfigMapName},
			},
		},
	}
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

func addRoles(
	instance *redhatcomv1alpha1.OpenStackAnsibleEE,
	h *helper.Helper,
	job *batchv1.Job,
	hashes map[string]string) {
	var roles []*redhatcomv1alpha1.Role
	roles = append(roles, instance.Spec.Role)
	d, err := yaml.Marshal(&roles)
	if err != nil {
		util.LogErrorForObject(h, err, err.Error(), instance)
	}

	var roleEnvVar corev1.EnvVar
	roleEnvVar.Name = "RUNNER_PLAYBOOK"
	roleEnvVar.Value = "\n" + string(d) + "\n\n"
	instance.Spec.Env = append(instance.Spec.Env, roleEnvVar)
	job.Spec.Template.Spec.Containers[0].Env = instance.Spec.Env
	hashes["roles"], err = calculateHash(string(d))
	if err != nil {
		h.GetLogger().Error(err, "Error calculating the hash")
	}
}

func addPlay(
	instance *redhatcomv1alpha1.OpenStackAnsibleEE,
	h *helper.Helper,
	job *batchv1.Job,
	hashes map[string]string) {
	var playEnvVar corev1.EnvVar
	var err error
	playEnvVar.Name = "RUNNER_PLAYBOOK"
	playEnvVar.Value = "\n" + instance.Spec.Play + "\n\n"
	instance.Spec.Env = append(instance.Spec.Env, playEnvVar)
	job.Spec.Template.Spec.Containers[0].Env = instance.Spec.Env
	hashes["play"], err = calculateHash(instance.Spec.Play)
	if err != nil {
		h.GetLogger().Error(err, "Error calculating the hash")
	}
}

func addPlaybook(
	instance *redhatcomv1alpha1.OpenStackAnsibleEE,
	h *helper.Helper,
	job *batchv1.Job,
	hashes map[string]string) {
	var playEnvVar corev1.EnvVar
	var err error
	playEnvVar.Name = "RUNNER_PLAYBOOK"
	playEnvVar.Value = "\n" + instance.Spec.Playbook + "\n\n"
	instance.Spec.Env = append(instance.Spec.Env, playEnvVar)
	job.Spec.Template.Spec.Containers[0].Env = instance.Spec.Env
	hashes["playbooks"], err = calculateHash(instance.Spec.Playbook)
	if err != nil {
		h.GetLogger().Error(err, "Error calculating the hash")
	}
}

func addInventory(
	instance *redhatcomv1alpha1.OpenStackAnsibleEE,
	h *helper.Helper,
	job *batchv1.Job,
	hashes map[string]string) {
	var invEnvVar corev1.EnvVar
	var err error
	invEnvVar.Name = "RUNNER_INVENTORY"
	invEnvVar.Value = "\n" + instance.Spec.Inventory + "\n\n"
	instance.Spec.Env = append(instance.Spec.Env, invEnvVar)
	job.Spec.Template.Spec.Containers[0].Env = instance.Spec.Env
	hashes["inventory"], err = calculateHash(instance.Spec.Inventory)
	if err != nil {
		h.GetLogger().Error(err, "Error calculating the hash")
	}
}

func addCmdLine(
	instance *redhatcomv1alpha1.OpenStackAnsibleEE,
	h *helper.Helper,
	job *batchv1.Job,
	hashes map[string]string) {
	var cmdLineEnvVar corev1.EnvVar
	var err error
	cmdLineEnvVar.Name = "RUNNER_CMDLINE"
	cmdLineEnvVar.Value = "\n" + instance.Spec.CmdLine + "\n\n"
	instance.Spec.Env = append(instance.Spec.Env, cmdLineEnvVar)
	job.Spec.Template.Spec.Containers[0].Env = instance.Spec.Env
	hashes["cmdline"], err = calculateHash(instance.Spec.CmdLine)
	if err != nil {
		h.GetLogger().Error(err, "Error calculating the hash")
	}
}

func calculateHash(envVar string) (string, error) {
	hash, err := util.ObjectHash(envVar)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func hashOfInputHashes(hashes map[string]string) (string, error) {
	var stringConcat string
	var err error
	if len(hashes) != 0 {
		for key, value := range hashes {
			// exclude hash defined by the job itself
			if key != "job" {
				stringConcat += stringConcat + value
			}
		}
	} else {
		stringConcat = ""
	}
	hash, err := util.ObjectHash(stringConcat)
	if err != nil {
		return hash, err
	}
	return hash, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackAnsibleEEReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&redhatcomv1alpha1.OpenStackAnsibleEE{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
