if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil {
    ko.Spec.Tags = rm.getTags(ctx, *ko.Status.ACKResourceMetadata.ARN)
}