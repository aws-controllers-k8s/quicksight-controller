{{/* 
  This hook is called after setting the output fields from the CreateDataSource API call.
  It checks if tags were specified and if so, marks the resource as not synced to trigger a requeue.
*/}}
import (
	corev1 "k8s.io/api/core/v1"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
)

if ko.Spec.Tags != nil {
    ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
}