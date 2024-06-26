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

//
// Generated by:
//
// operator-sdk create webhook --group ansibleee --version v1beta1 --kind OpenStackAnsibleEE --programmatic-validation --defaulting
//

package v1beta1

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// OpenStackAnsibleEEDefaults -
type OpenStackAnsibleEEDefaults struct {
	ContainerImageURL string
}

var openstackAnsibleEEDefaults OpenStackAnsibleEEDefaults

// log is for logging in this package.
var openstackansibleeelog = logf.Log.WithName("openstackansibleee-resource")

// SetupOpenStackAnsibleEEDefaults - initialize OpenStackAnsibleEE spec defaults for use with either internal or external webhooks
func SetupOpenStackAnsibleEEDefaults(defaults OpenStackAnsibleEEDefaults) {
	openstackAnsibleEEDefaults = defaults
	openstackansibleeelog.Info("OpenStackAnsibleEE defaults initialized", "defaults", defaults)
}

// SetupWebhookWithManager sets up the webhook with the Manager
func (r *OpenStackAnsibleEE) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-ansibleee-openstack-org-v1beta1-openstackansibleee,mutating=true,failurePolicy=fail,sideEffects=None,groups=ansibleee.openstack.org,resources=openstackansibleees,verbs=create;update,versions=v1beta1,name=mopenstackansibleee.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OpenStackAnsibleEE{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *OpenStackAnsibleEE) Default() {
	openstackansibleeelog.Info("default", "name", r.Name)

	r.Spec.Default()
}

// Default - set defaults for this OpenStackAnsibleEE spec
func (spec *OpenStackAnsibleEESpec) Default() {
	if spec.Image == "" {
		spec.Image = openstackAnsibleEEDefaults.ContainerImageURL
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-ansibleee-openstack-org-v1beta1-openstackansibleee,mutating=false,failurePolicy=fail,sideEffects=None,groups=ansibleee.openstack.org,resources=openstackansibleees,verbs=create;update,versions=v1beta1,name=vopenstackansibleee.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OpenStackAnsibleEE{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *OpenStackAnsibleEE) ValidateCreate() (admission.Warnings, error) {
	openstackansibleeelog.Info("validate create", "name", r.Name)
	var errors field.ErrorList

	errors = append(errors, r.Spec.ValidateCreate()...)
	if len(errors) != 0 {
		openstackansibleeelog.Info("validation failed", "name", r.Name)
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: "ansibleee.openstack.org", Kind: "OpenStakAnsibleEE"},
			r.Name,
			errors,
		)
	}
	return nil, nil
}

func (spec *OpenStackAnsibleEESpec) ValidateCreate() field.ErrorList {
	// Setting up input validation, including custom validators
	validate := validator.New()
	var errors field.ErrorList
	if valErr := validate.RegisterValidation("playbookContents", isPlaybook); valErr != nil {
		errors = append(errors,
			field.InternalError(
				field.NewPath("spec"),
				fmt.Errorf("error registering input validation")))
	}
	if valErr := validate.RegisterValidation("fqcn", isFQCN); valErr != nil {
		errors = append(errors,
			field.InternalError(
				field.NewPath("spec"),
				fmt.Errorf("error registering input validation")))
	}
	if len(spec.PlaybookContents) > 0 {
		valErr := validate.Var(spec.PlaybookContents, "playbookContents")
		if valErr != nil {
			errors = append(errors, field.Invalid(
				field.NewPath("spec.playbookContents"),
				spec.PlaybookContents,
				fmt.Sprintf(
					"error checking sanity of an inline playbook: %s %s",
					spec.PlaybookContents, valErr),
			))
		}
	} else if len(spec.Playbook) > 0 {
		// As we set "playbook.yaml" as default
		// we need to ensure that Play and Role are empty before addPlaybook
		if validate.Var(spec.Playbook, "fqcn") != nil {
			// First we check if the playbook isn't imported from a collection
			// if it is not we assume the variable holds a path.
			valErr := validate.Var(spec.Playbook, "filepath")
			if valErr != nil {
				errors = append(errors, field.Invalid(
					field.NewPath("spec.playbook"),
					spec.Playbook,
					fmt.Sprintf(
						"error checking sanity of playbook name/path: %s %s",
						spec.Playbook, valErr),
				))
			}
		}
	}

	for _, value := range spec.Env {
		if value.Name == "ANSIBLE_ENABLE_TASK_DEBUGGER" {
			errors = append(errors, field.Invalid(
				field.NewPath("spec.env"),
				spec.Env,
				"ansible-runner does not support ANSIBLE_ENABLE_TASK_DEBUGGER",
			))
		}
	}

	return errors
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *OpenStackAnsibleEE) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	openstackansibleeelog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *OpenStackAnsibleEE) ValidateDelete() (admission.Warnings, error) {
	openstackansibleeelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

// isPlaybook checks if the free form document has attributes of ansible playbook
// Specifically if it is a parsable yaml with list as a root element
func isPlaybook(document validator.FieldLevel) bool {
	var playbook []map[string]interface{}
	err := yaml.Unmarshal([]byte(document.Field().String()), &playbook)
	return err == nil
}

// isFQCN checks if the string matches regular expression of ansible FQCN
// The function doesn't check if the collection exists or is accessible
// function only accepts FQCNs as defined by
// https://old-galaxy.ansible.com/docs/contributing/namespaces.html#namespace-limitations
// Regex derived from ansible-lint rule
func isFQCN(name validator.FieldLevel) bool {
	pattern, compileErr := regexp.Compile(`^\w+(\.\w+){2,}$`)
	match := pattern.Match([]byte(name.Field().String()))
	return match && compileErr == nil
}
