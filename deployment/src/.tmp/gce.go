package main

import (
	"context"
	"fmt"
	"sync"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func getInstance(ctx context.Context, instancesClient *compute.InstancesClient, projectId string, zone string, instanceName string, wg *sync.WaitGroup, resultChan chan<- *Instance, errorChan chan<- error) {
	defer wg.Done()
	instanceId := getInstanceIdFromInstanceName(instanceName)

	instanceResponse, err := instancesClient.Get(ctx, &computepb.GetInstanceRequest{
		Project:  projectId,
		Zone:     zone,
		Instance: instanceId,
	})
	if err != nil {
		errorChan <- fmt.Errorf("error, while getting instance: %w", err)
		return
	}

	networkInterface := instanceResponse.GetNetworkInterfaces()
	internalIP := ""
	externalIP := ""
	for _, networkInterface := range networkInterface {
		if internalIP != "" && externalIP != "" {
			fmt.Printf("warning: multiple network interfaces found for an instance: %s. using the first one", instanceName)
			break
		}
		internalIP = networkInterface.GetNetworkIP()
		accessConfigs := networkInterface.GetAccessConfigs()
		for _, accessConfig := range accessConfigs {
			if externalIP != "" {
				fmt.Printf("warning: multiple external IPs found for an instance: %s. using the first one", instanceName)
				break
			}
			externalIP = accessConfig.GetNatIP()
		}
	}

	resultChan <- &Instance{
		InstanceName: instanceName,
		InstanceId:   instanceId,
		InternalIP:   internalIP,
		ExternalIP:   externalIP,
	}
}

func getInstances(ctx context.Context, projectId string, zone string, instanceGroupName string) ([]Instance, error) {
	computeClient, err := compute.NewInstanceGroupsRESTClient(ctx, option.WithCredentialsFile("service-account.json"))
	if err != nil {
		return nil, fmt.Errorf("error, while creating InstanceGroupsRESTClient: %w", err)
	}
	defer computeClient.Close()

	instanceNamesIterator := computeClient.ListInstances(ctx, &computepb.ListInstancesInstanceGroupsRequest{
		Project:       projectId,
		InstanceGroup: instanceGroupName,
		Zone:          zone,
	})

	var instanceNames []string
	for {
		resp, err := instanceNamesIterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error, while fetching instances: %w", err)
		}
		instanceNames = append(instanceNames, resp.GetInstance())
	}
	if len(instanceNames) == 0 {
		return nil, fmt.Errorf("error, while fetching instances: No instances found")
	}

	instancesClient, err := compute.NewInstancesRESTClient(ctx, option.WithCredentialsFile("service-account.json"))
	if err != nil {
		return nil, fmt.Errorf("error, while creating InstancesRESTClient: %w", err)
	}
	defer instancesClient.Close()

	var wg sync.WaitGroup
	instancesChan := make(chan *Instance, len(instanceNames))
	errorChan := make(chan error, len(instanceNames))

	for _, instanceName := range instanceNames {
		wg.Add(1)
		go getInstance(ctx, instancesClient, projectId, zone, instanceName, &wg, instancesChan, errorChan)
	}

	go func() {
		wg.Wait()
		close(instancesChan)
		close(errorChan)
	}()

	for err := range errorChan {
		if err != nil {
			return nil, err
		}
	}

	var instances []Instance
	for instance := range instancesChan {
		if instance != nil {
			instances = append(instances, *instance)
		}
	}

	return instances, nil
}
