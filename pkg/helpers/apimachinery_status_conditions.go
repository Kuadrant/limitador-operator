package helpers

import (
	"encoding/json"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeepCopyConditions copies the set of conditions
func DeepCopyConditions(conditions []metav1.Condition) []metav1.Condition {
	newConditions := make([]metav1.Condition, 0, len(conditions))
	for idx := range conditions {
		// copy
		newConditions = append(newConditions, *conditions[idx].DeepCopy())
	}
	return newConditions
}

// ConditionMarshal marshals the set of conditions as a JSON array, sorted by condition type.
func ConditionMarshal(conditions []metav1.Condition) ([]byte, error) {
	var condCopy []metav1.Condition
	condCopy = append(condCopy, conditions...)
	sort.Slice(condCopy, func(a, b int) bool {
		return condCopy[a].Type < condCopy[b].Type
	})
	return json.Marshal(condCopy)
}
