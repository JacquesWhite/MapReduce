#!/bin/bash

# Store the current directory
current_dir=$(pwd)

# Change to the project base
cd "$(dirname "$0")/../.." || exit

export REGION=us-central1
export PROJECT_ID=$(gcloud config get-value project)
export REPOSITORY_NAME=map-reduce
export MASTER_IMAGE_NAME=master
export WORKER_IMAGE_NAME=worker
export VERSION=latest

# Create registry
gcloud artifacts repositories create $REPOSITORY_NAME --repository-format=docker \
    --location=$REGION --description="Docker repository" \
    --project=$PROJECT_ID

# Configure Docker to authenticate to the registry
gcloud auth configure-docker $REGION-docker.pkg.dev

# Build the master image
docker build -t master -f docker/master/Dockerfile .

# Build the worker image
docker build -t worker -f docker/worker/Dockerfile .

# Build the upload image
cd src/upload
docker build -t upload -f Dockerfile .
cd ../../

MASTER_TAG=$REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY_NAME/$MASTER_IMAGE_NAME:$VERSION
WORKER_TAG=$REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY_NAME/$WORKER_IMAGE_NAME:$VERSION
UPLOAD_TAG=$REGION-docker.pkg.dev/$PROJECT_ID/$REPOSITORY_NAME/upload:$VERSION

# Tag images
docker tag master "$MASTER_TAG"
docker tag worker "$WORKER_TAG"
docker tag upload "$UPLOAD_TAG"

# Push images to the artifact registry on GCP
docker push "$MASTER_TAG"
docker push "$WORKER_TAG"
docker push "$UPLOAD_TAG"

cd $current_dir || exit