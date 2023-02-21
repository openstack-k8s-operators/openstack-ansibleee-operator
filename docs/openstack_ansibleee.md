
### Custom Resources

* [OpenStackAnsibleEE](#openstackansibleee)

### Sub Resources

* [Config](#config)
* [ImportRole](#importrole)
* [OpenStackAnsibleEEList](#openstackansibleeelist)
* [OpenStackAnsibleEESpec](#openstackansibleeespec)
* [Role](#role)
* [Task](#task)

#### Config

Config is a specification of where to mount a certain ConfigMap object

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| name | Name is the name of the ConfigMap that we want to mount | string | true |
| mountpath | MountPoint is the directory of the container where the ConfigMap will be mounted | string | true |

[Back to Custom Resources](#custom-resources)

#### ImportRole

ImportRole contains the actual rolename and tasks file name to execute

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| name |  | string | true |
| tasks_from |  | string | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackAnsibleEE

OpenStackAnsibleEE is the Schema for the openstackansibleees API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec |  | [OpenStackAnsibleEESpec](#openstackansibleeespec) | false |
| status |  | [OpenStackAnsibleEEStatus](#openstackansibleeestatus) | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackAnsibleEEList

OpenStackAnsibleEEList contains a list of OpenStackAnsibleEE

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][OpenStackAnsibleEE](#openstackansibleee) | true |

[Back to Custom Resources](#custom-resources)

#### OpenStackAnsibleEESpec

OpenStackAnsibleEESpec defines the desired state of OpenStackAnsibleEE

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| play | Play is the playbook contents that ansible will run on execution. If both Play and Roles are specified, Play takes precedence | string | false |
| playbook | Playbook is the playbook that ansible will run on this execution | string | false |
| image | Image is the container image that will execute the ansible command | string | false |
| args | Args are the command plus the playbook executed by the image. If args is passed, Playbook is ignored. | []string | false |
| name | Name is the name of the internal container inside the pod | string | false |
| env | Env is a list containing the environment variables to pass to the pod | []corev1.EnvVar | false |
| restartPolicy | RestartPolicy is the policy applied to the Job on whether it needs to restart the Pod. It can be \"OnFailure\" or \"Never\". | string | false |
| uid | UID is the userid that will be used to run the container. | int64 | false |
| inventory | Inventory is the inventory that the ansible playbook will use to launch the job. | string | false |
| extraMounts | ExtraMounts containing conf files and credentials | []storage.VolMounts | false |
| backoffLimit | BackoffLimimt allows to define the maximum number of retried executions. | *int32 | false |
| ttlSecondsAfterFinished | TTLSecondsAfterFinished specified the number of seconds the job will be kept in Kubernetes after completion. | *int32 | false |
| roles | Role is the description of an Ansible Role If both Play and Role are specified, Play takes precedence | [Role](#role) | false |

[Back to Custom Resources](#custom-resources)

#### Role

Role describes the format of an ansible playbook destinated to run roles

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| name |  | string | false |
| hosts |  | string | false |
| strategy |  | string | false |
| any_errors_fatal |  | bool | false |
| become |  | bool | false |
| gather_facts |  | bool | false |
| tasks |  | [][Task](#task) | true |

[Back to Custom Resources](#custom-resources)

#### Task

Task describes a task centered exclusively in running import_role

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| name |  | string | true |
| import_role |  | [ImportRole](#importrole) | true |
| vars |  | string | false |
| tags |  | []string | false |

[Back to Custom Resources](#custom-resources)
