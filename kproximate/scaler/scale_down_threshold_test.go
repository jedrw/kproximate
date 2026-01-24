package scaler

import (
	"testing"

	"github.com/jedrw/kproximate/config"
	"github.com/jedrw/kproximate/kubernetes"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSelectScaleDownTargetWithThreshold(t *testing.T) {
	tests := []struct {
		name                    string
		minNodeLoadForScaleDown float64
		nodeLoads               map[string]float64
		expectError             bool
		expectedNode            string
	}{
		{
			name:                    "Threshold disabled (0) - always selects node",
			minNodeLoadForScaleDown: 0,
			nodeLoads: map[string]float64{
				"node-1": 0.45,
				"node-2": 0.42, // Lowest
				"node-3": 0.48,
			},
			expectError:  false,
			expectedNode: "node-2",
		},
		{
			name:                    "All nodes above threshold - rejects scale down",
			minNodeLoadForScaleDown: 0.3,
			nodeLoads: map[string]float64{
				"node-1": 0.45,
				"node-2": 0.42, // Lowest but > 0.3
				"node-3": 0.48,
			},
			expectError:  true,
			expectedNode: "",
		},
		{
			name:                    "One node below threshold - selects it",
			minNodeLoadForScaleDown: 0.3,
			nodeLoads: map[string]float64{
				"node-1": 0.45,
				"node-2": 0.25, // Lowest and < 0.3
				"node-3": 0.48,
			},
			expectError:  false,
			expectedNode: "node-2",
		},
		{
			name:                    "Exactly at threshold - rejects",
			minNodeLoadForScaleDown: 0.3,
			nodeLoads: map[string]float64{
				"node-1": 0.45,
				"node-2": 0.30, // Exactly at threshold
				"node-3": 0.48,
			},
			expectError:  true,
			expectedNode: "",
		},
		{
			name:                    "Very low utilization - selects node",
			minNodeLoadForScaleDown: 0.3,
			nodeLoads: map[string]float64{
				"node-1": 0.15,
				"node-2": 0.05, // Very low
				"node-3": 0.20,
			},
			expectError:  false,
			expectedNode: "node-2",
		},
		{
			name:                    "High threshold (0.5) - rejects moderately loaded nodes",
			minNodeLoadForScaleDown: 0.5,
			nodeLoads: map[string]float64{
				"node-1": 0.60,
				"node-2": 0.45, // Lowest but < 0.5
				"node-3": 0.55,
			},
			expectError:  false,
			expectedNode: "node-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock nodes
			var nodes []*apiv1.Node
			allocatedResources := make(map[string]kubernetes.AllocatedResources)

			for nodeName, load := range tt.nodeLoads {
				node := &apiv1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: nodeName,
						Labels: map[string]string{
							"kpNodeTemplateName": "test-template",
						},
					},
				}
				nodes = append(nodes, node)

				// Convert combined load back to CPU and Memory
				// Assuming equal split for simplicity
				cpuLoad := load / 2.0
				memoryLoad := load / 2.0

				allocatedResources[nodeName] = kubernetes.AllocatedResources{
					Cpu:    cpuLoad * 4.0,  // 4 cores per node
					Memory: memoryLoad * 4096.0 * 1024 * 1024, // 4GB per node in bytes
				}
			}

			// Create scaler with mock
			scaler := ProxmoxScaler{
				Kubernetes: &kubernetes.KubernetesMock{
					Nodes:              nodes,
					AllocatedResources: allocatedResources,
				},
				config: config.KproximateConfig{
					KpNodeCores:             4,
					KpNodeMemory:            4096,
					KpNodeTemplateName:      "test-template",
					MinNodeLoadForScaleDown: tt.minNodeLoadForScaleDown,
				},
			}

			scaleEvent := ScaleEvent{}
			err := scaler.selectScaleDownTarget(&scaleEvent)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if scaleEvent.NodeName != tt.expectedNode {
					t.Errorf("Expected node %s but got %s", tt.expectedNode, scaleEvent.NodeName)
				}
			}
		})
	}
}

func TestSelectScaleDownTargetWithTemplateDrift(t *testing.T) {
	// Template drift should take priority over threshold
	nodes := []*apiv1.Node{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-1",
				Labels: map[string]string{
					"kpNodeTemplateName": "old-template", // Drifted
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-2",
				Labels: map[string]string{
					"kpNodeTemplateName": "test-template",
				},
			},
		},
	}

	allocatedResources := map[string]kubernetes.AllocatedResources{
		"node-1": {Cpu: 3.0, Memory: 3000}, // High load
		"node-2": {Cpu: 0.5, Memory: 500},  // Low load
	}

	scaler := ProxmoxScaler{
		Kubernetes: &kubernetes.KubernetesMock{
			Nodes:              nodes,
			AllocatedResources: allocatedResources,
		},
		config: config.KproximateConfig{
			KpNodeCores:             4,
			KpNodeMemory:            4096,
			KpNodeTemplateName:      "test-template",
			MinNodeLoadForScaleDown: 0.3,
		},
	}

	scaleEvent := ScaleEvent{}
	err := scaler.selectScaleDownTarget(&scaleEvent)

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	if scaleEvent.NodeName != "node-1" {
		t.Errorf("Expected drifted node-1 to be selected, got: %s", scaleEvent.NodeName)
	}
}

func TestSelectScaleDownTargetNoNodes(t *testing.T) {
	scaler := ProxmoxScaler{
		Kubernetes: &kubernetes.KubernetesMock{
			Nodes:              []*apiv1.Node{},
			AllocatedResources: map[string]kubernetes.AllocatedResources{},
		},
		config: config.KproximateConfig{
			KpNodeCores:             4,
			KpNodeMemory:            4096,
			MinNodeLoadForScaleDown: 0.3,
		},
	}

	scaleEvent := ScaleEvent{}
	err := scaler.selectScaleDownTarget(&scaleEvent)

	if err == nil {
		t.Errorf("Expected error for no nodes but got none")
	}
}
