{{/* 
  This hook is called after setting the output fields from the CreateDataSource API call.
  It checks if tags were specified and if so, marks the resource as not synced to trigger a requeue.
*/}}
if ko.Spec.Tags != nil {
    ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
}