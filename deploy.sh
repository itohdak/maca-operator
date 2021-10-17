#!/bin/bash

set -e

make docker-build
kind load docker-image controller:latest
make install
make deploy
kubectl delete -f config/samples/maca_v1alpha1_loadtestmanager.yaml
kubectl rollout restart -n macaoperator-system deployment macaoperator-controller-manager
kubectl apply -f config/samples/maca_v1alpha1_loadtestmanager.yaml

