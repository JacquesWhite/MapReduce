# MapReduce
Map Reduce project for **Engineering Distributed Infrastructure** class at University of Warsaw

## Set up application on Google Cloud Platform with Kubernetes

### Using script
1. Create a new project on Google Cloud Platform
2. Install Google Cloud SDK
   - [Google Cloud SDK](https://cloud.google.com/sdk/docs/install)
3. Authenticate with Google Cloud SDK
   - `gcloud auth login`
   - `gcloud config set project PROJECT_ID`
4. Run the turnup script
   - `src/scripts/setup_app_on_gcloud.sh`

### Useful links
- [Workloads (pods)](https://console.cloud.google.com/kubernetes/workload/overview)
- [GKE Storage](https://console.cloud.google.com/kubernetes/persistentvolumeclaims)
- [NFS instance](https://console.cloud.google.com/filestore/instances)

## Using application deployed to GEK on Google Cloud Platform
1. Use GKE feature of port forwarding to a specific pod:
    - `kubectl port-forward mapreduce-upload 8000:8000`
    - Open browser and go to `localhost:8000`
2. After running `setup_app_on_gcloud.sh` script You should see the message:
    - ` MapReduce is now available at: http://<Address>` - just go to that address
3. Check the address of the service:
    - `kubectl get svc mapreduce-upload-loadbalancer -o jsonpath='{.status.loadBalancer.ingress[0].ip}'`