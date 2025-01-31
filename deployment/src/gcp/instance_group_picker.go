package gcp

import "github.com/charmbracelet/huh"

type instanceGroupItem struct {
	name string
	zone string
}

func (i instanceGroupItem) Title() string       { return i.name + " (" + i.zone + ")" }
func (i instanceGroupItem) Description() string { return "" }
func (i instanceGroupItem) FilterValue() string { return i.name }

type InstancePickerModel struct {
	gceInstances   []GceInstance
	instancePicker *huh.Form
}
