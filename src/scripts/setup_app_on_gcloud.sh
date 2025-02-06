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
export FILESTORE_PATH="mapreduceshare"

gcloud components install gke-gcloud-auth-plugin

# Create a GKE cluster
gcloud container clusters create-auto $CLUSTER_NAME --region $REGION

# Get the credentials for the cluster
gcloud container clusters get-credentials $CLUSTER_NAME --region $REGION

# Set up the Google Cloud NFS instance
gcloud filestore instances create $FILESTORE_NAME \
  --project=$PROJECT_ID \
  --zone=$FILESTORE_ZONE \
  --tier=STANDARD \
  --file-share=name=$FILESTORE_PATH,capacity=1TB \
  --network=name="default"

export FILESTORE_IP=$(
  gcloud filestore instances describe $FILESTORE_NAME --zone=$FILESTORE_ZONE --format='value(networks.ipAddresses[0])'
)

# Optional step, if you want to update the Docker images used by pods
./docker_upload.sh

# GKE stuff
kubectl create -f ./../../k8s-deployment/storage/filestore-storageclass.yml
kubectl create -f ./../../k8s-deployment/storage/pvc.yml
envsubst < ./../../k8s-deployment/storage/pv.yml | kubectl apply -f -

envsubst < ./../../k8s-deployment/master-service.yml | kubectl apply -f -
envsubst < ./../../k8s-deployment/master-statefulset.yml | kubectl apply -f -
envsubst < ./../../k8s-deployment/worker-deployment.yml | kubectl apply -f -
envsubst < ./../../k8s-deployment/upload-service.yml | kubectl apply -f -
envsubst < ./../../k8s-deployment/upload-statefulset.yml | kubectl apply -f -

cd $current_dir || exit