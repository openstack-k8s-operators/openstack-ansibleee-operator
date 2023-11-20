
### Custom Resources

* [OpenStackAnsibleEE](#openstackansibleee)

### Sub Resources

* [Config](#config)
* [OpenStackAnsibleEEList](#openstackansibleeelist)
* [OpenStackAnsibleEESpec](#openstackansibleeespec)
* [OpenStackAnsibleEEStatus](#openstackansibleeestatus)

#### Config

Config is a specification of where to mount a certain ConfigMap object

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| name | Name is the name of the ConfigMap that we want to mount | string | true |
| mountpath | MountPoint is the directory of the container where the ConfigMap will be mounted | string | true |

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
| play | Play is an inline playbook contents that ansible will run on execution. If both Play and Roles are specified, Play takes precedence | string | false |
| playbook | Playbook is the playbook that ansible will run on this execution, accepts path or FQN from collection | string | false |
| image | Image is the container image that will execute the ansible command | string | false |
| args | Args are the command plus the playbook executed by the image. If args is passed, Playbook is ignored. | []string | false |
| name | Name is the name of the internal container inside the pod | string | false |
| envConfigMapName | EnvConfigMapName is the name of the k8s config map that contains the ansible env variables | string | false |
| env | Env is a list containing the environment variables to pass to the pod | []corev1.EnvVar | false |
| restartPolicy | RestartPolicy is the policy applied to the Job on whether it needs to restart the Pod. It can be \"OnFailure\" or \"Never\". RestartPolicy default: OnFailure | string | false |
| preserveJobs | PreserveJobs - do not delete jobs after they finished e.g. to check logs PreserveJobs default: true | bool | false |
| uid | UID is the userid that will be used to run the container. | int64 | false |
| inventory | Inventory is the inventory that the ansible playbook will use to launch the job. | string | false |
| extraMounts | ExtraMounts containing conf files and credentials | []storage.VolMounts | false |
| backoffLimit | BackoffLimit allows to define the maximum number of retried executions (defaults to 6). | *int32 | false |
| networkAttachments | NetworkAttachments is a list of NetworkAttachment resource names to expose the services to the given network | []string | false |
| cmdLine | CmdLine is the command line passed to ansible-runner | string | false |
| initContainers | InitContainers allows the passing of an array of containers that will be executed before the ansibleee execution itself | []corev1.Container | false |
| serviceAccountName | ServiceAccountName allows to specify what ServiceAccountName do we want the ansible execution run with. Without specifying, it will run with default serviceaccount | string | false |
| dnsConfig | DNSConfig allows to specify custom dnsservers and search domains | *corev1.PodDNSConfig | false |
| debug | Debug run the pod in debug mode without executing the ansible-runner commands | bool | true |
| extraVars | Extra vars to be passed to ansible process during execution. This can be used to override default values in plays. | map[string]json.RawMessage | false |

[Back to Custom Resources](#custom-resources)

#### OpenStackAnsibleEEStatus

OpenStackAnsibleEEStatus defines the observed state of OpenStackAnsibleEE

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| hash | Map of hashes to track e.g. job status | map[string]string | false |
| conditions | Conditions | condition.Conditions | false |
| networkAttachments | NetworkAttachments status of the deployment pods | map[string][]string | false |
| JobStatus | JobStatus status of the executed job (Pending/Running/Succeeded/Failed) | string | false |

[Back to Custom Resources](#custom-resources)
