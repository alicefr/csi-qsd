#!/bin/bash
TAG=latest
IMAGE_DRIVER_NAME="qsd/driver"
IMAGE_QSD_NAME="qsd/qsd"
IMAGE_DRIVER="${IMAGE_DRIVER_NAME}:${TAG}"
IMAGE_QSD="${IMAGE_QSD_NAME}:${TAG}"
CLUSTER=k8s-qsd
kubectl delete -f deployment/driver.yaml

set -ex
kind load docker-image --name ${CLUSTER}  ${IMAGE_DRIVER}
kind load docker-image --name ${CLUSTER} ${IMAGE_QSD}
kubectl apply -f deployment
