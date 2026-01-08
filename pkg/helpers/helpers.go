package helpers

import (
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DeleteTagAnnotation       = "limitador.kuadrant.io/delete"
	LabelKeyApp               = "app"
	LabelKeyLimitadorResource = "limitador-resource"
	LabelValueLimitador       = "limitador"
)

func ObjectInfo(obj client.Object) string {
	return fmt.Sprintf("%s/%s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
}

func TagObjectToDelete(obj client.Object) {
	// Add custom annotation
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
		obj.SetAnnotations(annotations)
	}
	annotations[DeleteTagAnnotation] = "true"
}

func IsObjectTaggedToDelete(obj client.Object) bool {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false
	}

	annotation, ok := annotations[DeleteTagAnnotation]
	return ok && annotation == "true"
}

func MergeMapStringString(existing *map[string]string, desired map[string]string) bool {
	if existing == nil {
		return false
	}
	if *existing == nil {
		*existing = map[string]string{}
	}

	// for each desired key value set, e.g. labels
	// check if it's present in existing. if not add it to existing.
	// e.g. preserving existing labels while adding those that are in the desired set.
	modified := false
	for desiredKey, desiredValue := range desired {
		if existingValue, exists := (*existing)[desiredKey]; !exists || existingValue != desiredValue {
			(*existing)[desiredKey] = desiredValue
			modified = true
		}
	}
	return modified
}
