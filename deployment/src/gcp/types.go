package gcp

type InstanceURL string
type InstanceId string
type IpAddress string

type GceInstance struct {
	InstanceURL InstanceURL
	InstanceId  InstanceId
	InternalIP  IpAddress
	ExternalIP  IpAddress
}
type ProjectURL string
type ProjectId string

type GceProject struct {
	ProjectURL ProjectURL
	ProjectId  ProjectId
}
