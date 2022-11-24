#!/bin/bash
set -e

echo "UNDEPLOY"
make undeploy

echo "BUILD"
make generate
make manifests
make build

echo "DEPLOY"
kl create -f config/crd/bases/redhat.com_ansibleees.yaml
./bin/manager
