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

	yaml "gopkg.in/yaml.v3"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	redhatcomv1alpha1 "github.com/openstack-k8s-operators/ansibleee-operator/api/v1alpha1"
)

// AnsibleEEReconciler reconciles a AnsibleEE object
type AnsibleEEReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Log     logr.Logger
	Scheme  *runtime.Scheme
}

// +kubebuilder:rbac:groups=redhat.com,resources=ansibleees,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=redhat.com,resources=ansibleees/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=redhat.com,resources=ansibleees/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;
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

	instance, err := r.getAnsibleeeInstance(ctx, req)
	if err != nil || instance.Name == "" {
		return ctrl.Result{}, err
	}

	//configMapVars := make(map[string]env.Setter)
	//r.configMapForAnsibleEE(ctx, instance, helper, &configMapVars)

	if len(instance.Spec.Inventory) > 0 {
		foundCM := &corev1.ConfigMap{}
		err = r.Get(ctx, types.NamespacedName{Name: "inventory-configmap", Namespace: instance.Namespace}, foundCM)
		if err != nil && errors.IsNotFound(err) {
			// Define a new configmap
			cm := r.createConfigMapInventory(instance)
			fmt.Printf("Creating a new ConfigMap: ConfigMap.Namespace %s ConfigMap.Name %s\n", cm.Namespace, "inventory-configmap")
			err = r.Create(ctx, cm)
			if err != nil {
				fmt.Println(err.Error())
				return ctrl.Result{}, err
			}
			fmt.Println("configmap created successfully - return and requeue")
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			fmt.Println(err.Error())
			//log.Error(err, "Failed to get Job")
			return ctrl.Result{}, err
		}
	}

	if len(instance.Spec.Play) > 0 {
		foundCM := &corev1.ConfigMap{}
		err = r.Get(ctx, types.NamespacedName{Name: "playbook-configmap", Namespace: instance.Namespace}, foundCM)
		if err != nil && errors.IsNotFound(err) {
			// Define a new configmap
			cm := r.createConfigMapPlay(instance)
			fmt.Printf("Creating a new ConfigMap: ConfigMap.Namespace %s ConfigMap.Name %s\n", cm.Namespace, "playbook-configmap")
			err = r.Create(ctx, cm)
			if err != nil {
				fmt.Println(err.Error())
				return ctrl.Result{}, err
			}
			fmt.Println("configmap created successfully - return and requeue")
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			fmt.Println(err.Error())
			//log.Error(err, "Failed to get Job")
			return ctrl.Result{}, err
		}
	}

	// Check if the job already exists, if not create a new one
	foundJob := &batchv1.Job{}
	err = r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, foundJob)
	if err != nil && errors.IsNotFound(err) {
		// Define a new job
		job, err := r.jobForAnsibleEE(instance)
		if err != nil {
			return ctrl.Result{}, err
		}
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

	return ctrl.Result{}, nil
}

func (r *AnsibleEEReconciler) getAnsibleeeInstance(ctx context.Context, req ctrl.Request) (*redhatcomv1alpha1.AnsibleEE, error) {
	// Fetch the AnsibleEE instance
	instance := &redhatcomv1alpha1.AnsibleEE{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			fmt.Println("AnsibleEE resource not found. Ignoring since object must be deleted")
			//log.Info("AnsibleEE resource not found. Ignoring since object must be deleted")
			return &redhatcomv1alpha1.AnsibleEE{}, nil
		}
		// Error reading the object - requeue the request.
		fmt.Println(err.Error())
		//log.Error(err, "Failed to get AnsibleEE")
		return &redhatcomv1alpha1.AnsibleEE{}, err
	}

	return instance, nil
}

// jobForAnsibleEE returns a ansibleee Job object
func (r *AnsibleEEReconciler) jobForAnsibleEE(instance *redhatcomv1alpha1.AnsibleEE) (*batchv1.Job, error) {
	ls := labelsForAnsibleEE(instance.Name)

	args := instance.Spec.Args

	if len(args) == 0 {
		if len(instance.Spec.Playbook) == 0 {
			args = []string{"ansible-runner", "run", "/runner", "-p", "playbook.yaml"}
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
					Labels: ls,
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

	addInventory(instance, job)
	addRoles(instance, job)
	addMounts(instance, job)

	// Set AnsibleEE instance as the owner and controller
	err := ctrl.SetControllerReference(instance, job, r.Scheme)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (r *AnsibleEEReconciler) createConfigMapInventory(instance *redhatcomv1alpha1.AnsibleEE) *corev1.ConfigMap {
	data := map[string]string{
		"inventory.yaml": instance.Spec.Inventory,
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inventory-configmap",
			Namespace: instance.Namespace,
		},
		Data: data,
	}

	return cm
}

func (r *AnsibleEEReconciler) createConfigMapPlay(instance *redhatcomv1alpha1.AnsibleEE) *corev1.ConfigMap {
	data := map[string]string{
		"playbook.yaml": instance.Spec.Play,
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "playbook-configmap",
			Namespace: instance.Namespace,
		},
		Data: data,
	}

	return cm
}

// labelsForAnsibleEE returns the labels for selecting the resources
// belonging to the given ansibleee CR name.
func labelsForAnsibleEE(name string) map[string]string {
	return map[string]string{"app": "ansibleee", "ansibleee_cr": name}
}

func addMounts(instance *redhatcomv1alpha1.AnsibleEE, job *batchv1.Job) {
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume

	if len(instance.Spec.Inventory) > 0 {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "inventory",
			MountPath: "/runner/inventory/inventory.yaml",
			SubPath:   "inventory.yaml",
		})
		volumes = append(volumes, corev1.Volume{
			Name: "inventory",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "inventory-configmap",
					},
				},
			},
		})
	}

	if len(instance.Spec.Play) > 0 {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "playbook",
			MountPath: "/runner/projects/playbook.yaml",
			SubPath:   "playbook.yaml",
		})
		volumes = append(volumes, corev1.Volume{
			Name: "playbook",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "playbook-configmap",
					},
				},
			},
		})
	}

	for i := 0; i < len(instance.Spec.Configs); i++ {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      instance.Spec.Configs[i].Name,
			MountPath: instance.Spec.Configs[i].MountPath,
		})
		volumes = append(volumes, corev1.Volume{
			Name: instance.Spec.Configs[i].Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: instance.Spec.Configs[i].Name,
					},
				},
			},
		})
	}
	job.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts
	job.Spec.Template.Spec.Volumes = volumes
}

func addRoles(instance *redhatcomv1alpha1.AnsibleEE, job *batchv1.Job) {
	var roles []*redhatcomv1alpha1.Role
	roles = append(roles, &instance.Spec.Role)
	d, err := yaml.Marshal(&roles)
	if err != nil {
		fmt.Println(err.Error())
	}

	var roleEnvVar corev1.EnvVar
	roleEnvVar.Name = "RUNNER_STANDALONE_ROLE"
	roleEnvVar.Value = "\n" + string(d) + "\n\n"
	instance.Spec.Env = append(instance.Spec.Env, roleEnvVar)
	job.Spec.Template.Spec.Containers[0].Env = instance.Spec.Env
}

func addInventory(instance *redhatcomv1alpha1.AnsibleEE, job *batchv1.Job) {
	var invEnvVar corev1.EnvVar
	invEnvVar.Name = "RUNNER_INVENTORY"
	invEnvVar.Value = "\n" + instance.Spec.Inventory + "\n\n"
	instance.Spec.Env = append(instance.Spec.Env, invEnvVar)
	job.Spec.Template.Spec.Containers[0].Env = instance.Spec.Env
}

// SetupWithManager sets up the controller with the Manager.
func (r *AnsibleEEReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&redhatcomv1alpha1.AnsibleEE{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
