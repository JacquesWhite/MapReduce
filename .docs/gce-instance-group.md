# gce instance groups

## gce instance group resize

```bash
gcloud beta compute instance-groups managed update worker-instance-group \
    --project=subtle-circlet-421210 \
    --zone=northamerica-south1-a \
    --size=10 \
    --update-policy-type=opportunistic \
    --default-action-on-vm-failure=repair \
    --no-force-update-on-repair \
    --standby-policy-mode=manual \
    --suspended-size=0 \
    --stopped-size=0 \
    --standby-policy-initial-delay=0 \
    --list-managed-instances-results=pageless \
&& \
gcloud beta compute instance-groups managed rolling-action start-update worker-instance-group \
    --project=subtle-circlet-421210 \
    --zone=northamerica-south1-a \
    --type=opportunistic \
    --version=template=projects/subtle-circlet-421210/regions/northamerica-south1/instanceTemplates/worker-instance-e2-mexico \
&& \
gcloud beta compute instance-groups managed set-autoscaling worker-instance-group \
    --project=subtle-circlet-421210 \
    --zone=northamerica-south1-a \
    --mode=off \
    --min-num-replicas=0 \
    --max-num-replicas=10 \
    --target-cpu-utilization=0.6 \
    --cpu-utilization-predictive-method=none \
    --cool-down-period=60
```

## run command on all instances

```bash
for i z in $(gcloud compute instances list --format='value(name, zone)'); do
  gcloud compute ssh $i --command="ls -lah" --zone=$z;
done
```
