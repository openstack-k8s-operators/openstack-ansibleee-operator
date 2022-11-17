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
Once the operator has been deployed succesfully to the openshift/kubernetes cluster, you can see it in action by creating a new "ansibleee" CR. An example can be found at examples/ansibleee-test.yaml.
```
kubectl apply -f examples/ansibleee-test.yaml
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
oc create -f example/ansibleee-test.yaml
```
The operator will create a ansible pod and run the playbook. It will
then move to a completed state.
```
$ oc get pods | grep ansible
ansibleee-test-q4pt9         0/1     Completed   0          24m
$
```
To see the result of the playbook run, use `oc logs`.
```
oc logs $(oc get pods | grep ansible | awk {'print $1'})
```
