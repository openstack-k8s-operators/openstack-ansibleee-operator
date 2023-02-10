# OpenStack Ansible EE operator

An operator to deploy and run an OpenStack Ansible Execution Environment container on Openshift

## Build and deploy

It uses operator-sdk to build and run.

To build and push to a docker repository

```bash
make docker-build docker-push IMG="<your image name>"
```

To deploy in to the cluster

```bash
make deploy IMG="<your image name>"
```

To undeploy it from the cluster

```bash
make undeploy
```

## Use

Once the operator has been deployed succesfully to the openshift/kubernetes cluster, you can see it in action by creating a new "ansibleee" CR.

There are some examples on the examples directory.

The first one is ansibleee-playbook-local.yaml. This wil execute locally the playbook "test.yaml", which will run some checks on the container where ansible-runner is being executed.

```bash
oc apply -f examples/openstack-ansibleee-playbook-local.yaml
```

There are other examples that also execute locally the playbook "test.yaml", but that serve as extraMounts demonstration: ansibleee-extravolumes.yaml and ansibleee-extravolumes_2_secret.yaml that need the secrets ceph-secret-example.yaml and ceph-secret-example2.yaml created:

```bash
oc apply -f ceph-secret-example.yaml
oc apply -f ceph-secret-example2.yaml
oc apply -f examples/openstack-ansibleee-extravolumes.yaml
```

There are also a number of examples that feature remote execution. By default, all of them expect a compute node to be available in 10.0.0.4, adjust the inventory accordingly for your environment. This setup is compatible with the libvirt development environment deployment described in [libvirt_podified_standalone](https://gitlab.cee.redhat.com/rhos-upgrades/data-plane-adoption-dev/-/blob/main/libvirt_podified_standalone.md).

The first remote example is ansibleee-playbook.yaml. This runs one of the standalone playbooks that is included in the default image.

To access an external node, you need to provide the ssh private key so ansible can connect to the node. This is being expected to be provided by a "ssh-key-secret" Secret with this format:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ssh-key-secret
  namespace: openstack
data:
  ssh-privatekey:  3390 bytes                                                                                       │
  ssh-publickey:   750 bytes
```

Once the key has been created, the CR should run the deploy-tripleo-os-configure.yml playbook on the external node:

```bash
oc apply -f examples/openstack-ansibleee-playbook.yaml
```

The second remote example is ansibleee-role.yaml, which will run a certain number of tasks from specific standalone roles:

```bash
oc apply -f examples/openstack-ansibleee-role.yaml
```

And the last remote example is ansibleee-play.yaml, which will run a CR-defined playbook using an inventory stored in a ConfigMap.

```bash
oc apply -f examples/inventory-configmap.yaml
oc apply -f examples/openstack-ansibleee-play.yaml
```

## Example Development Cycle

The following has been verified on
[openshift-local](https://developers.redhat.com/products/openshift-local/overview).

The Makefile assumes you have docker installed. If you're using
podman, then adjust accordingly (e.g. symlink docker to podman).

Create the CRD managed by the operator. This must be deleted and re-created any time the api changes.

```bash
oc create -f config/crd/bases/ansibleee.openstack.org_openstackansibleees.yaml
```

Build and run a local copy of the OpenStack Ansible Execution Environment operator.

```bash
make generate
make manifests
make build
./bin/manager
```

Once the operator is running, create the examle CR to run the test playbook.

```bash
oc create -f examples/openstack-ansibleee-playbook-local.yaml
```

The operator will create a ansible pod and run the playbook. It will
then move to a completed state.

```bash
$ oc get pods | grep ansible
ansibleee-playbook-local-q4pt9         0/1     Completed   0          24m
```

To see the result of the playbook run, use `oc logs`.

```bash
oc logs $(oc get pods | grep ansible | awk {'print $1'})
```

## Using openstack-ansibleee-operator with EDPM Ansible

When the openstack-ansibleee-operator spawns a job ansible execution environment crafted image
can use playbooks and roles contained in its image.

An openstack-ansibleee-runner image is hosted at
[quay.io/openstack-k8s-operators/openstack-ansibleee-runner](https://quay.io/openstack-k8s-operators/openstack-ansibleee-runner)
which contains [edpm-ansible](https://github.com/openstack-k8s-operators/edpm-ansible).
The following commands may be used to inspect the content.

```bash
podman pull quay.io/openstack-k8s-operators/openstack-ansibleee-runner:latest
IMAGE_ID=$(podman images --filter reference=openstack-ansibleee-runner:latest --format "{{.Id}}")
podman run $IMAGE_ID ls -l
```

The container is built by a github actions from a [Dockerfile](https://github.com/openstack-k8s-operators/edpm-ansible/blob/main/openstack_ansibleee/Dockerfile) in the [edpm-ansible](https://github.com/openstack-k8s-operators/edpm-ansible) repository.
