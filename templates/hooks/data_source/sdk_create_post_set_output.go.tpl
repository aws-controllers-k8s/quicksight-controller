{{/* 
  This hook is called after setting the output fields from the CreateDataSource API call.
  It sets the Status field from CreationStatus and marks resource as not synced if tags were specified.
*/}}
if ko.Spec.Tags != nil {
    ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
}
