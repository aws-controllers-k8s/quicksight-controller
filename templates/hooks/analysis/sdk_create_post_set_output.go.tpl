{{/* Template for the SDK Create Post Set Output hook for Analysis */}}
{{/* This hook is called after the output of the Create operation is set */}}
{{/* It checks if Tags are set and marks the resource as not synced */}}

if ko.Spec.Tags != nil {
    ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
}