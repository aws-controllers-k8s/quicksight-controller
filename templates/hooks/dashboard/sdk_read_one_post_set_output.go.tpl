	if resp.Dashboard.Version != nil {
		if resp.Dashboard.Version.VersionNumber != nil {
			ko.Status.VersionNumber = resp.Dashboard.Version.VersionNumber
		}
		if resp.Dashboard.Version.Status != "" {
			ko.Status.VersionStatus = aws.String(string(resp.Dashboard.Version.Status))
		}
	}
	ko.Spec.Tags, err = getTags(ctx, string(*ko.Status.ACKResourceMetadata.ARN), rm.sdkapi, rm.metrics)
	if err != nil {
		return &resource{ko}, err
	}
