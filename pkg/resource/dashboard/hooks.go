// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package dashboard

import (
	"context"
	"fmt"

	"github.com/aws-controllers-k8s/quicksight-controller/pkg/sync"
	"github.com/aws-controllers-k8s/runtime/pkg/metrics"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	"github.com/aws/aws-sdk-go-v2/aws"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/quicksight"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/quicksight/types"
)

var syncTags = sync.Tags
var getTags = sync.GetTags

// dashboardVersionReady returns true if the latest version is in a terminal
// successful state and can be published.
func dashboardVersionReady(r *resource) bool {
	if r.ko.Status.VersionStatus == nil {
		return false
	}
	status := *r.ko.Status.VersionStatus
	return status == string(svcsdktypes.ResourceStatusCreationSuccessful) || status == string(svcsdktypes.ResourceStatusUpdateSuccessful)
}

// requeueWaitVersionReady returns a RequeueNeededAfter indicating the
// dashboard version is not yet ready to be published.
func requeueWaitVersionReady(r *resource) *ackrequeue.RequeueNeededAfter {
	status := "unknown"
	if r.ko.Status.VersionStatus != nil {
		status = *r.ko.Status.VersionStatus
	}
	return ackrequeue.NeededAfter(
		fmt.Errorf("dashboard version in '%s' state, waiting to publish", status),
		ackrequeue.DefaultRequeueAfterDuration,
	)
}

// getLatestDashboardVersion calls ListDashboardVersions (with pagination)
// and returns the version number and status of the version with the highest
// version number.
func getLatestDashboardVersion(
	ctx context.Context,
	sdkapi *svcsdk.Client,
	m *metrics.Metrics,
	awsAccountID *string,
	dashboardID *string,
) (versionNumber *int64, versionStatus *string, err error) {
	var latest *svcsdktypes.DashboardVersionSummary
	var nextToken *string
	for {
		input := &svcsdk.ListDashboardVersionsInput{
			AwsAccountId: awsAccountID,
			DashboardId:  dashboardID,
			MaxResults:   aws.Int32(100),
			NextToken:    nextToken,
		}
		resp, err := sdkapi.ListDashboardVersions(ctx, input)
		m.RecordAPICall("READ_MANY", "ListDashboardVersions", err)
		if err != nil {
			return nil, nil, err
		}
		for i := range resp.DashboardVersionSummaryList {
			v := &resp.DashboardVersionSummaryList[i]
			if latest == nil || (v.VersionNumber != nil && *v.VersionNumber > *latest.VersionNumber) {
				latest = v
			}
		}
		if resp.NextToken == nil || *resp.NextToken == "" {
			break
		}
		nextToken = resp.NextToken
	}
	if latest == nil {
		return nil, nil, nil
	}
	status := string(latest.Status)
	return latest.VersionNumber, &status, nil
}
