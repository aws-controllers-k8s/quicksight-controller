{{- if .CRD.Spec.Tags }}
if ko.Spec.Tags != nil {
    ackcondition.SetSynced(&resource{ko}, corev1.ConditionFalse, nil, nil)
}
{{- end }}