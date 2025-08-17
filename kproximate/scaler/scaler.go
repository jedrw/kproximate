package scaler

import (
	"context"

	"github.com/jedrw/kproximate/proxmox"
)

const (
	ScaleUpQueueName   = "scaleUpEvents"
	ScaleDownQueueName = "scaleDownEvents"
	ReplaceQueueName   = "replaceEvents"
)

type Scaler interface {
	RequiredScaleUpEvents(numCurrentEvents int) ([]*ScaleEvent, error)
	SelectTargetHosts(scaleEvents []*ScaleEvent) error
	ScaleUp(ctx context.Context, scaleEvent *ScaleEvent) error
	NumReadyNodes() (int, error)
	NumNodes() (int, error)
	AssessScaleDown() (*ScaleEvent, error)
	ScaleDown(ctx context.Context, scaleEvent *ScaleEvent) error
	DeleteNode(ctx context.Context, kpNodeName string) error
	GetResourceStatistics() (ResourceStatistics, error)
	AssessNodeDrift() (*ScaleEvent, error)
	ReplaceNode(ctx context.Context, replaceEvent *ScaleEvent) (string, error)
}

type ScaleEvent struct {
	NodeName   string
	TargetHost proxmox.HostInformation
}

type AllocatedResources struct {
	Cpu    float64
	Memory float64
}

type AllocatableResources struct {
	Cpu    float64
	Memory float64
}

type ResourceStatistics struct {
	Allocatable AllocatableResources
	Allocated   AllocatedResources
}
