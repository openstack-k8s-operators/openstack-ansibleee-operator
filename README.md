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

The second one is ansibleee-playbook.yaml. This one has a more evolved playbook with an inventory that actually points to an external node. It is compatible with the libvirt development environment deployment described in [libvirt_podified_standalone](https://gitlab.cee.redhat.com/rhos-upgrades/data-plane-adoption-dev/-/blob/main/libvirt_podified_standalone.md).

Remember that to access an external node, you need to provide the ssh private key so ansible can connect to the node. This is being provided by the "key-configmap" ConfigMap (it will be a secret once the lib-common [extramounts system](https://github.com/openstack-k8s-operators/ansibleee-operator/pull/6) is integrated) with this format:
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: key-configmap
  namespace: openstack
data:
  ssh_key: |
    -----BEGIN OPENSSH PRIVATE KEY-----
    b3BlbnNzaC1rZX...
    -----END OPENSSH PRIVATE KEY-----
```

Once the key has been created, the CR should run the deploy-tripleo-os-configure.yml playbook on the external node:
```
oc apply -f examples/ansibleee-playbook.yaml
```




Modify the yaml to suit your needs.

## Example Development Cycle

The following has been verified on
[openshift-local](https://developers.redhat.com/products/openshift-local/overview).

The Makefile assumes you have docker installed. If you're using
podman, then adjust accordingly (e.g. symlink docker to podman).

The example CR, examples/ansibleee-test.yaml, assumes the following
`ConfigMap` resources have been created so create them first.
```
oc create -f example/swift-configmap.yaml
oc create -f example/test-configmap-1.yaml
oc create -f example/test-configmap-2.yaml
```
Create the CRD managed by the operator.
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
oc create -f example/ansibleee-playbook-local.yaml
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
