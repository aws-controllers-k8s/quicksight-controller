	// A VersionNumber diff means UpdateDashboard created a new draft version.
	// We must publish it before updating Status.VersionNumber on the desired
	// resource. If we set VersionNumber before a successful publish, a
	// subsequent reconcile would see no diff and skip the publish, leaving
	// the dashboard stuck on the old published version.
	if delta.DifferentAt("Status.VersionNumber") {
		if !dashboardVersionReady(latest) {
			return desired, requeueWaitVersionReady(latest)
		}
		_, pubErr := rm.sdkapi.UpdateDashboardPublishedVersion(ctx, &svcsdk.UpdateDashboardPublishedVersionInput{
			AwsAccountId:  desired.ko.Spec.AWSAccountID,
			DashboardId:   desired.ko.Spec.ID,
			VersionNumber: latest.ko.Status.VersionNumber,
		})
		rm.metrics.RecordAPICall("UPDATE", "UpdateDashboardPublishedVersion", pubErr)
		if pubErr != nil {
			return desired, pubErr
		}
		// Safe to propagate now — the new version is published.
		desired.ko.Status.VersionNumber = latest.ko.Status.VersionNumber
		desired.ko.Status.VersionStatus = latest.ko.Status.VersionStatus
	}
	if delta.DifferentAt("Spec.Tags") {
		arn := string(*latest.ko.Status.ACKResourceMetadata.ARN)
		err = syncTags(
			ctx,
			desired.ko.Spec.Tags, latest.ko.Spec.Tags,
			&arn, convertToOrderedACKTags, rm.sdkapi, rm.metrics,
		)
		if err != nil {
			return desired, err
		}
	}
	if !delta.DifferentExcept("Spec.Tags", "Status.VersionNumber") {
		return desired, nil
	}

