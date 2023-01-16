#!/bin/bash
#set -e

echo "UNDEPLOY"
oc delete -f config/crd/bases/redhat.com_openstackansibleees.yaml

echo "BUILD"
make generate
make manifests
make build

echo "DEPLOY"
oc apply -f config/crd/bases/redhat.com_openstackansibleees.yaml
./bin/manager
