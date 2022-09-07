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
Once the operator has been deployed succesfully to the openshift/kubernetes cluster, you can see it in action by creating a new "ansibleee" CR. An example can be found at examples/ansibleee-jlarriba.yaml.
```
kubectl apply -f examples/ansibleee-jlarriba.yaml
```

Modify the yaml to suit your needs.
