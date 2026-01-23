{{/* 
  This hook is called after setting the output fields from the DescribeDataSource API call.
  It retrieves the tags for the resource and sets them in the Spec.Tags field.
*/}}
if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil {
    ko.Spec.Tags = rm.getTags(ctx, *ko.Status.ACKResourceMetadata.ARN)
}