#!/bin/bash
TAG=latest
IMAGE_DRIVER_NAME="qsd/driver"
IMAGE_QSD_NAME="qsd/qsd"
IMAGE_DRIVER="${IMAGE_DRIVER_NAME}:${TAG}"
IMAGE_QSD="${IMAGE_QSD_NAME}:${TAG}"
CLUSTER=k8s-qsd
kubectl delete -f deployment/driver.yaml
docker exec -ti k8s-qsd-control-plane crictl rmi ${IMAGE_DRIVER}
docker exec -ti k8s-qsd-control-plane crictl rmi ${IMAGE_QSD}
kubectl  delete po -l name=qsd
set -ex
kind load docker-image --name ${CLUSTER}  ${IMAGE_DRIVER}
kind load docker-image --name ${CLUSTER} ${IMAGE_QSD}
kubectl apply -f deployment/namespace.yaml
kubectl apply -f deployment/qsd-ds.yaml
kubectl apply -f deployment/driver.yaml
kubectl apply -f deployment/snapshotclass.yaml

# Set context
kubectl config set-context csi --namespace csi-qsd --user kind-k8s-qsd --cluster kind-k8s-qsd
kubectl config use-context csi
