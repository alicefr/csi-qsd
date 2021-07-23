#!/bin/bash

kind create cluster --name k8s-qsd --config hack/cluster/kind.yaml

# Apply VolumeSnapshot CRDs and controller
# The version needs to match with the exernal-snapshotter sidecar
kubectl apply -f deployment/snapshotter-${SNAPSHOTTER_VERSION}
