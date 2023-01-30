#!/bin/bash
set -e

echo "UNDEPLOY"
make undeploy

echo "BUILD & PUSH"
make docker-buildx IMG="quay.io/openstack-k8s-operators/openstack-ansibleee-operator"

echo "DEPLOY"
make deploy IMG="quay.io/openstack-k8s-operators/openstack-ansibleee-operator"
