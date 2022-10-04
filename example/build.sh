#!/bin/bash
set -e

echo "UNDEPLOY"
make undeploy

echo "BUILD & PUSH"
make docker-build docker-push IMG="quay.io/jlarriba/ansibleee-operator"

echo "DEPLOY"
make deploy IMG="quay.io/jlarriba/ansibleee-operator"
