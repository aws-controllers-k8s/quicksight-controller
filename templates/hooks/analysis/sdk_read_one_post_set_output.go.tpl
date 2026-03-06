{{/* Template for the SDK Read One Post Set Output hook for Analysis */}}
{{/* This hook is called after the output of the Read operation is set */}}
{{/* It retrieves the tags for the Analysis resource */}}

if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil {
    ko.Spec.Tags = rm.getTags(ctx, *ko.Status.ACKResourceMetadata.ARN)
}