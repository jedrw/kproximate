package kubernetes

import (
	"testing"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAffinityMismatchDetection(t *testing.T) {
	tests := []struct {
		name                    string
		conditionMessage        string
		expectAffinityMismatch  bool
		expectSelectorMismatch  bool
	}{
		{
			name:                   "Node affinity mismatch",
			conditionMessage:       "0/6 nodes are available: 1 node(s) had untolerated taint, 5 node(s) didn't match Pod's node affinity/selector.",
			expectAffinityMismatch: true,
			expectSelectorMismatch: false,
		},
		{
			name:                   "Node selector mismatch",
			conditionMessage:       "0/3 nodes are available: 3 node(s) didn't match node selector.",
			expectAffinityMismatch: false,
			expectSelectorMismatch: true,
		},
		{
			name:                   "Insufficient CPU",
			conditionMessage:       "0/3 nodes are available: 3 Insufficient cpu.",
			expectAffinityMismatch: false,
			expectSelectorMismatch: false,
		},
		{
			name:                   "Combined affinity and selector",
			conditionMessage:       "0/6 nodes are available: 2 node(s) didn't match Pod's node affinity/selector, 4 node(s) didn't match node selector.",
			expectAffinityMismatch: true,
			expectSelectorMismatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pod",
				},
				Status: apiv1.PodStatus{
					Conditions: []apiv1.PodCondition{
						{
							Type:    apiv1.PodScheduled,
							Status:  apiv1.ConditionFalse,
							Reason:  apiv1.PodReasonUnschedulable,
							Message: tt.conditionMessage,
						},
					},
				},
			}

			hasAffinityMismatch := false
			hasSelectorMismatch := false

			for _, condition := range pod.Status.Conditions {
				if isUnschedulable(condition) {
					if containsString(condition.Message, "didn't match Pod's node affinity/selector") {
						hasAffinityMismatch = true
					}
					if containsString(condition.Message, "didn't match node selector") {
						hasSelectorMismatch = true
					}
				}
			}

			if hasAffinityMismatch != tt.expectAffinityMismatch {
				t.Errorf("Expected affinity mismatch=%v, got=%v", tt.expectAffinityMismatch, hasAffinityMismatch)
			}
			if hasSelectorMismatch != tt.expectSelectorMismatch {
				t.Errorf("Expected selector mismatch=%v, got=%v", tt.expectSelectorMismatch, hasSelectorMismatch)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
