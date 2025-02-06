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