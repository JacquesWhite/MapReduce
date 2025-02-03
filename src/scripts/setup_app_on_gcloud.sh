# Store the current directory
current_dir=$(pwd)

# Change to the project base
cd "$(dirname "$0")" || exit

export CLUSTER_NAME=map-reduce-cluster
export REGION=us-central1
export REPOSITORY_NAME=map-reduce
export PROJECT_ID=$(gcloud config get-value project)

gcloud components install gke-gcloud-auth-plugin

# Create a GKE cluster
gcloud container clusters create-auto $CLUSTER_NAME --region $REGION

# Get the credentials for the cluster
gcloud container clusters get-credentials $CLUSTER_NAME --region $REGION

./docker_upload.sh

# GKE stuff
kubectl create -f ./../../k8s-deployment/storage/filestore-storageclass.yml
kubectl create -f ./../../k8s-deployment/storage/pvc.yml

envsubst < ./../../k8s-deployment/master-service.yml | kubectl apply -f -
envsubst < ./../../k8s-deployment/master-statefulset.yml | kubectl apply -f -
envsubst < ./../../k8s-deployment/worker-deployment.yml | kubectl apply -f -

cd $current_dir || exit