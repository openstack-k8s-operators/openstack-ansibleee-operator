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
	"regexp"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"context"

	"github.com/go-logr/logr"
	"github.com/go-playground/validator/v10"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	helper "github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	job "github.com/openstack-k8s-operators/lib-common/modules/common/job"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	util "github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	ansibleeev1 "github.com/openstack-k8s-operators/openstack-ansibleee-operator/api/v1beta1"
)

const (
	ansibleeeJobType       = "ansibleee"
	ansibleeeInputHashName = "input"
)

// OpenStackAnsibleEEReconciler reconciles a OpenStackAnsibleEE object
type OpenStackAnsibleEEReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Scheme  *runtime.Scheme
}

// GetLogger returns a logger object with a prefix of "controller.name" and additional controller context fields
func (r *OpenStackAnsibleEEReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("OpenStackAnsibleEE")
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
	Log := r.GetLogger(ctx)

	instance, err := r.getOpenStackAnsibleeeInstance(ctx, req)
	if err != nil || instance.Name == "" {
		return ctrl.Result{}, err
	}

	helper, err := helper.NewHelper(
		instance,
		r.Client,
		r.Kclient,
		r.Scheme,
		Log,
	)

	if err != nil {
		// helper might be nil, so can't use util.LogErrorForObject since it requires helper as first arg
		Log.Error(err, fmt.Sprintf("Unable to acquire helper for  OpenStackAnsibleEE %s", instance.Name))
		return ctrl.Result{}, err
	}

	// Always patch the instance status when exiting this function so we can
	// persist any changes.
	defer func() {
		// update the overall status condition if service is ready
		if instance.IsReady() {
			instance.Status.Conditions.MarkTrue(condition.ReadyCondition, ansibleeev1.AnsibleExecutionJobReadyMessage)
		} else {
			// something is not ready so reset the Ready condition
			instance.Status.Conditions.MarkUnknown(
				condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage)
			// and recalculate it based on the state of the rest of the conditions
			instance.Status.Conditions.Set(instance.Status.Conditions.Mirror(condition.ReadyCondition))
		}

		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			Log.Error(_err, "PatchInstance error")
			_err = err
			return
		}
	}()

	// Initialize Status
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = condition.Conditions{}

		cl := condition.CreateList(
			condition.UnknownCondition(ansibleeev1.AnsibleExecutionJobReadyCondition, condition.InitReason, ansibleeev1.AnsibleExecutionJobInitMessage),
		)

		instance.Status.Conditions.Init(&cl)

		// Register overall status immediately to have an early feedback e.g.
		// in the cli
		return ctrl.Result{}, nil
	}

	// Initialize Status fields
	util.InitMap(&instance.Status.Hash)
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
	jobDef, err := r.jobForOpenStackAnsibleEE(ctx, instance, helper, serviceAnnotations)
	if err != nil {
		return ctrl.Result{}, err
	}

	configMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: instance.Spec.EnvConfigMapName, Namespace: instance.Namespace}, configMap)
	if err != nil && !errors.IsNotFound(err) {
		Log.Error(err, err.Error())
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
			ansibleeev1.AnsibleExecutionJobReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			ansibleeev1.AnsibleExecutionJobWaitingMessage))
		instance.Status.JobStatus = ansibleeev1.JobStatusRunning
		return ctrlResult, nil
	}

	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			ansibleeev1.AnsibleExecutionJobReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			ansibleeev1.AnsibleExecutionJobErrorMessage,
			err.Error()))
		instance.Status.JobStatus = ansibleeev1.JobStatusFailed
		return ctrl.Result{}, err
	}

	if ansibleeeJob.HasChanged() {
		instance.Status.Hash[ansibleeeJobType] = ansibleeeJob.GetHash()
		Log.Info(fmt.Sprintf("AnsibleEE CR '%s' - Job %s hash added - %s", instance.Name, jobDef.Name, instance.Status.Hash[ansibleeeJobType]))
	}

	instance.Status.Conditions.MarkTrue(ansibleeev1.AnsibleExecutionJobReadyCondition, ansibleeev1.AnsibleExecutionJobReadyMessage)
	instance.Status.JobStatus = ansibleeev1.JobStatusSucceeded

	Log.Info(fmt.Sprintf("Reconciled AnsibleEE '%s' successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *OpenStackAnsibleEEReconciler) getOpenStackAnsibleeeInstance(ctx context.Context, req ctrl.Request) (*ansibleeev1.OpenStackAnsibleEE, error) {
	Log := r.GetLogger(ctx)

	// Fetch the OpenStackAnsibleEE instance
	instance := &ansibleeev1.OpenStackAnsibleEE{}

	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			Log.Info("OpenStackAnsibleEE resource not found. Ignoring since object must be deleted")
			return &ansibleeev1.OpenStackAnsibleEE{}, nil
		}
		// Error reading the object - requeue the request.
		Log.Error(err, err.Error())
		return instance, err
	}

	return instance, nil
}

// jobForOpenStackAnsibleEE returns a openstackansibleee Job object
func (r *OpenStackAnsibleEEReconciler) jobForOpenStackAnsibleEE(ctx context.Context,
	instance *ansibleeev1.OpenStackAnsibleEE,
	h *helper.Helper,
	annotations map[string]string) (*batchv1.Job, error) {
	Log := r.GetLogger(ctx)
	labels := instance.GetObjectMeta().GetLabels()

	// Setting up input validation, including custom validators
	validate := validator.New()

	if valErr := validate.RegisterValidation("play", isPlay); valErr != nil {
		return nil, fmt.Errorf("error registering input validation")
	}
	if valErr := validate.RegisterValidation("fqcn", isFQCN); valErr != nil {
		return nil, fmt.Errorf("error registering input validation")
	}

	ls := labelsForOpenStackAnsibleEE(instance.Name, labels)

	args := instance.Spec.Args

	playbook := instance.Spec.Playbook
	if len(args) == 0 {
		if len(playbook) == 0 {
			playbook = "playbook.yaml"
		}
		args = []string{"ansible-runner", "run", "/runner", "-p", playbook}
	}

	// ansible runner identifier
	// if the flag is set we use resource name as an argument
	// https://ansible-runner.readthedocs.io/en/stable/intro/#artifactdir
	if !(util.StringInSlice("-i", args) || util.StringInSlice("--ident", args)) {
		identifier := instance.Name
		args = append(args, []string{"-i", identifier}...)
	}

	// Override args list if we are in a debug mode
	if instance.Spec.Debug {
		args = []string{"sleep", "1d"}
		Log.Info(fmt.Sprintf("Instance %s will be running in debug mode.", instance.Name))
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
			Name:        instance.Name,
			Namespace:   instance.Namespace,
			Annotations: annotations,
			Labels:      ls,
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
		setRunnerEnvVar(instance, h, "RUNNER_INVENTORY", instance.Spec.Inventory, "inventory", job, hashes)
	}

	if len(instance.Spec.Play) > 0 {
		valErr := validate.Var(instance.Spec.Play, "play")
		if valErr != nil {
			return nil, fmt.Errorf(
				"error checking sanity of an inline play: %s %w",
				instance.Spec.Play, valErr)
		}
		setRunnerEnvVar(instance, h, "RUNNER_PLAYBOOK", instance.Spec.Play, "play", job, hashes)
	} else if len(playbook) > 0 {
		// As we set "playbook.yaml" as default
		// we need to ensure that Play and Role are empty before addPlaybook
		if validate.Var(playbook, "fqcn") != nil {
			// First we check if the playbook isn't imported from a collection
			// if it is not we assume the variable holds a path.
			valErr := validate.Var(playbook, "filepath")
			if valErr != nil {
				return nil, fmt.Errorf(
					"error checking sanity of playbook name/path: %s %w",
					playbook, valErr)
			}
		}

		setRunnerEnvVar(instance, h, "RUNNER_PLAYBOOK", playbook, "playbooks", job, hashes)
	}

	if len(instance.Spec.CmdLine) > 0 && !instance.Spec.Debug {
		// RUNNER_CMDLINE environment variable should only be set
		// if the operator isn't running in a debug mode.
		setRunnerEnvVar(instance, h, "RUNNER_CMDLINE", instance.Spec.CmdLine, "cmdline", job, hashes)
	}
	if len(labels["deployIdentifier"]) > 0 {
		hashes["deployIdentifier"] = labels["deployIdentifier"]
	}

	addMounts(instance, job)

	// if we have any extra vars for ansible to use set them in the RUNNER_EXTRA_VARS
	if len(instance.Spec.ExtraVars) > 0 {
		parsedExtraVars := ""
		// unmarshal nested data structures
		for variable, val := range instance.Spec.ExtraVars {
			var tmp interface{}
			err := yaml.Unmarshal(val, &tmp)
			if err != nil {
				return nil, err
			}
			parsedExtraVars += fmt.Sprintf("%s: %s\n", variable, tmp)
		}
		setRunnerEnvVar(instance, h, "RUNNER_EXTRA_VARS", parsedExtraVars, "extraVars", job, hashes)
	}

	hashPodSpec(h, podSpec, hashes)

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
func labelsForOpenStackAnsibleEE(name string, labels map[string]string) map[string]string {
	ls := map[string]string{
		"app":                   "openstackansibleee",
		"job-name":              name,
		"openstackansibleee_cr": name,
		"osaee":                 "true",
	}
	for key, val := range labels {
		ls[key] = val
	}
	return ls
}

func addEnvFrom(instance *ansibleeev1.OpenStackAnsibleEE, job *batchv1.Job) {
	job.Spec.Template.Spec.Containers[0].EnvFrom = []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: instance.Spec.EnvConfigMapName},
			},
		},
	}
}

func addMounts(instance *ansibleeev1.OpenStackAnsibleEE, job *batchv1.Job) {
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

func hashPodSpec(
	h *helper.Helper,
	podSpec corev1.PodSpec,
	hashes map[string]string,
) {
	var err error
	spec, _ := podSpec.Marshal()
	hashes["podspec"], err = calculateHash(string(spec))

	if err != nil {
		h.GetLogger().Error(err, "Error calculating the PodSpec hash")
	}
}

// set value of runner environment variable and compute the hash
func setRunnerEnvVar(instance *ansibleeev1.OpenStackAnsibleEE,
	helper *helper.Helper,
	varName string,
	varValue string,
	hashType string,
	job *batchv1.Job,
	hashes map[string]string) {

	var envVar corev1.EnvVar
	var err error
	envVar.Name = varName
	envVar.Value = "\n" + varValue + "\n\n"
	job.Spec.Template.Spec.Containers[0].Env = append(job.Spec.Template.Spec.Containers[0].Env, envVar)
	hashes[hashType], err = calculateHash(varValue)
	if err != nil {
		helper.GetLogger().Error(err, "Error calculating the hash")
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
	var err error
	var hash string
	var builder strings.Builder
	for key, value := range hashes {
		// exclude hash defined by the job itself
		if key != "job" {
			_, err := builder.WriteString(value)
			if err != nil {
				return hash, err
			}
		}
	}
	hash, err = util.ObjectHash(builder.String())
	if err != nil {
		return hash, err
	}
	return hash, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenStackAnsibleEEReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ansibleeev1.OpenStackAnsibleEE{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}

// isPlay checks if the free form document has attributes of ansible play
// Specifically if it is a parsable yaml with list as a root element
func isPlay(document validator.FieldLevel) bool {
	var play []map[string]interface{}
	err := yaml.Unmarshal([]byte(document.Field().String()), &play)
	return err == nil
}

// isFQCN checks if the string matches regular expression of ansible FQCN
// The function doesn't check if the collection exists or is accessible
// function only accepts FQCNs as defined by
// https://galaxy.ansible.com/docs/contributing/namespaces.html#namespace-limitations
// Regex derive
func isFQCN(name validator.FieldLevel) bool {
	pattern, compileErr := regexp.Compile(`^\w+(\.\w+){2,}$`)
	match := pattern.Match([]byte(name.Field().String()))
	return match && compileErr == nil
}
