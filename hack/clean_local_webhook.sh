#!/bin/bash
set -ex

oc delete validatingwebhookconfiguration/vopenstackansibleee.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mopenstackansibleee.kb.io --ignore-not-found
