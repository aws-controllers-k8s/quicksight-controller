	if resp.Dashboard.Version != nil {
		if resp.Dashboard.Version.VersionNumber != nil {
			ko.Status.VersionNumber = resp.Dashboard.Version.VersionNumber
		}
		if resp.Dashboard.Version.Status != "" {
			ko.Status.VersionStatus = aws.String(string(resp.Dashboard.Version.Status))
		}
		if resp.Dashboard.Version.ThemeArn != nil {
			ko.Spec.ThemeARN = resp.Dashboard.Version.ThemeArn
		}
		if resp.Dashboard.Version.SourceEntityArn != nil {
			if ko.Spec.SourceEntity == nil {
				ko.Spec.SourceEntity = &svcapitypes.DashboardSourceEntity{}
			}
			if ko.Spec.SourceEntity.SourceTemplate == nil {
				ko.Spec.SourceEntity.SourceTemplate = &svcapitypes.DashboardSourceTemplate{}
			}
			ko.Spec.SourceEntity.SourceTemplate.ARN = resp.Dashboard.Version.SourceEntityArn
		}
		if resp.Dashboard.Version.Description != nil {
			ko.Spec.VersionDescription = resp.Dashboard.Version.Description
		}
	}
	ko.Spec.Tags, err = getTags(ctx, string(*ko.Status.ACKResourceMetadata.ARN), rm.sdkapi, rm.metrics)
	if err != nil {
		return &resource{ko}, err
	}
