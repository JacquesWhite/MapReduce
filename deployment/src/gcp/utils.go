package gcp

import "strings"

func getInstanceIdFromInstanceURL(instanceURL InstanceURL) InstanceId {
	parts := strings.Split(string(instanceURL), "/")
	return InstanceId(parts[len(parts)-1])
}

func getProjectIdFromProjectURL(projectURL ProjectURL) ProjectId {
	parts := strings.Split(string(projectURL), "/")
	return ProjectId(parts[len(parts)-1])
}
