# ansibleee-operator
An operator to deploy and run an Ansible Execution Environment container on Openshit

# Build and deploy
It uses operator-sdk to build and run.

To build and push to a docker repository
```
make docker-build docker-push IMG="<your image name>"
```

To deploy in to the cluster
```
make deploy IMG="<your image name>"
```

To undeploy it from the cluster
```
make undeploy
```

# Use
Once the operator has been deployed succesfully to the openshift/kubernetes cluster, you can see it in action by creating a new "ansibleee" CR. 

There are some examples on the examples directory.

The first one is ansibleee-playbook-local.yaml. This wil execute locally the playbook "test.yaml", which will run some checks on the container where ansible-runner is being executed.
```
oc apply -f examples/ansibleee-playbook-local.yaml
```

There are other examples that also execute locally the playbook "test.yaml", but that serve as extraMounts demonstration: ansibleee-extravolumes.yaml and ansibleee-extravolumes_2_secret.yaml that need the secrets ceph-secret-example.yaml and ceph-secret-example2.yaml created:
```
oc apply -f ceph-secret-example.yaml
oc apply -f ceph-secret-example2.yaml
oc apply -f examples/ansibleee-extravolumes.yaml
```

There are also a number of examples that feature remote execution. By default, all of them expect a compute node to be available in 10.0.0.4, adjust the inventory accordingly for your environment. This setup is compatible with the libvirt development environment deployment described in [libvirt_podified_standalone](https://gitlab.cee.redhat.com/rhos-upgrades/data-plane-adoption-dev/-/blob/main/libvirt_podified_standalone.md).

The first remote example is ansibleee-playbook.yaml. This runs one of the standalone playbooks that is included in the default image.

To access an external node, you need to provide the ssh private key so ansible can connect to the node. This is being expected to be provided by a "ssh-key-secret" Secret with this format:
```
apiVersion: v1
kind: Secret
metadata:
  name: ssh-key-secret
  namespace: openstack
data:
  ssh-privatekey:  3390 bytes                                                                                       │
│ ssh-publickey:   750 bytes
```

Once the key has been created, the CR should run the deploy-tripleo-os-configure.yml playbook on the external node:
```
oc apply -f examples/ansibleee-playbook.yaml
```

The second remote example is ansibleee-role.yaml, which will run a certain number of tasks from specific standalone roles:
```
oc apply -f examples/ansibleee-role.yaml
```

The third remote example is ansibleee-play.yaml, which will run a CR-defined playbook using an inventory stored in a ConfigMap.
```
oc apply -f examples/inventory-configmap.yaml
oc apply -f examples/ansibleee-play.yaml
```

The fourth remote example is ansibleee-plugin.yaml, which will run a custom inventory plugin using `cloudguruab.edpm_plugin` collection using two configmaps; 1. edpm-configmap.yaml and 2. plugin-configmap.yaml to set the appropriate output given in each respective ConfigMap.
```
oc apply -f examples/plugin-configmap.yaml
oc apply -f examples/edpm-configmap.yaml
```

## Example Development Cycle

The following has been verified on
[openshift-local](https://developers.redhat.com/products/openshift-local/overview).

The Makefile assumes you have docker installed. If you're using
podman, then adjust accordingly (e.g. symlink docker to podman).

Create the CRD managed by the operator. This must be deleted and re-created any time the api changes.
```
oc create -f config/crd/bases/redhat.com_ansibleees.yaml
```
Build and run a local copy of the Ansible Execution Environment operator.
```
make generate
make manifests
make build
./bin/manager
```
Once the operator is running, create the examle CR to run the test playbook.
```
oc create -f examples/ansibleee-playbook-local.yaml
```
The operator will create a ansible pod and run the playbook. It will
then move to a completed state.
```
$ oc get pods | grep ansible
ansibleee-playbook-local-q4pt9         0/1     Completed   0          24m
$
```
To see the result of the playbook run, use `oc logs`.
```
oc logs $(oc get pods | grep ansible | awk {'print $1'})
```
