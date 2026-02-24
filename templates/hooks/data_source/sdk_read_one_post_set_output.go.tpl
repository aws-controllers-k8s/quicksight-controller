{{/* 
  This hook is called after setting the output fields from the DescribeDataSource API call.
  It retrieves the tags for the resource and sets them in the Spec.Tags field.
  Note: Field unwrapping is handled by output_wrapper_field_path in generator.yaml
*/}}
if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil {
    ko.Spec.Tags = rm.getTags(ctx, *ko.Status.ACKResourceMetadata.ARN)
}