	latestVersionNumber, latestVersionStatus, lvErr := getLatestDashboardVersion(ctx, rm.sdkapi, rm.metrics, ko.Spec.AWSAccountID, ko.Spec.ID)
	if lvErr != nil {
		return &resource{ko}, lvErr
	}
	ko.Status.VersionNumber = latestVersionNumber
	if latestVersionStatus != nil {
		ko.Status.VersionStatus = latestVersionStatus
	}
	ko.Spec.Tags, err = getTags(ctx, string(*ko.Status.ACKResourceMetadata.ARN), rm.sdkapi, rm.metrics)
	if err != nil {
		return &resource{ko}, err
	}

