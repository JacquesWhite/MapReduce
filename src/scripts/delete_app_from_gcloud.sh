#!/bin/bash
# Cleanup script for GCP deploy.

# Store the current directory
current_dir=$(pwd)

# Google Cloud CLI: https://cloud.google.com/sdk/docs/install

# Change to the project base
cd "$(dirname "$0")" || exit

export CLUSTER_NAME=map-reduce-cluster
export REGION=us-central1
export REPOSITORY_NAME=map-reduce
export PROJECT_ID=$(gcloud config get-value project)
export FILESTORE_NAME=filestore-map-reduce
export FILESTORE_ZONE=$REGION-a

# Delete the services and deployments
kubectl delete service mapreduce-master
kubectl delete statefulset mapreduce-master
kubectl delete service mapreduce-upload
kubectl delete statefulset mapreduce-upload
kubectl delete service mapreduce-upload-loadbalancer
kubectl delete deployment mapreduce-workers
kubectl delete hpa mapreduce-workers-autoscaler

# Delete the log-based metrics (everything needs separate confirmation)
gcloud logging metrics delete mapreduce-master-critical
gcloud logging metrics delete mapreduce-master-info
gcloud logging metrics delete mapreduce-worker-critical
gcloud logging metrics delete mapreduce-worker-info

# Delete filestore
kubectl delete pvc mapreduce-pvc

# Delete whole filestore instance (needs separate confirmation)
gcloud filestore instances delete $FILESTORE_NAME \
  --project=$PROJECT_ID \
  --zone=$FILESTORE_ZONE \
  --force

# Delete the repository (needs separate confirmation)
gcloud artifacts repositories delete $REPOSITORY_NAME --location=$REGION --project=$PROJECT_ID

# Delete the GKE cluster (needs separate confirmation)
gcloud container clusters delete $CLUSTER_NAME --region $REGION