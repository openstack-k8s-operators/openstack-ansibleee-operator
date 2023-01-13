#!/bin/bash
#set -e

echo "UNDEPLOY"
oc delete -f config/crd/bases/redhat.com_ansibleees.yaml

echo "BUILD"
make generate
make manifests
make build

echo "DEPLOY"
oc apply -f config/crd/bases/redhat.com_ansibleees.yaml
./bin/manager
