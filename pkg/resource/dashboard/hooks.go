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
	"strings"

	"github.com/aws-controllers-k8s/quicksight-controller/pkg/sync"
	"github.com/aws-controllers-k8s/runtime/pkg/metrics"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	svcsdk "github.com/aws/aws-sdk-go-v2/service/quicksight"
	svcsdktypes "github.com/aws/aws-sdk-go-v2/service/quicksight/types"
)

var syncTags = sync.Tags
var getTags = sync.GetTags

// dashboardVersionReady calls DescribeDashboard for the given version number
// and returns (true, status) if that version is in a terminal successful state
// and can be published, or (false, status) otherwise.
func dashboardVersionReady(
	ctx context.Context,
	sdkapi *svcsdk.Client,
	m *metrics.Metrics,
	r *resource,
	versionNumber *int64,
) (bool, string) {
	if versionNumber == nil {
		return false, ""
	}
	resp, err := sdkapi.DescribeDashboard(ctx, &svcsdk.DescribeDashboardInput{
		AwsAccountId:  r.ko.Spec.AWSAccountID,
		DashboardId:   r.ko.Spec.ID,
		VersionNumber: versionNumber,
	})
	m.RecordAPICall("READ_ONE", "DescribeDashboard", err)
	if err != nil || resp.Dashboard == nil || resp.Dashboard.Version == nil {
		return false, ""
	}
	status := string(resp.Dashboard.Version.Status)
	ready := resp.Dashboard.Version.Status == svcsdktypes.ResourceStatusCreationSuccessful || resp.Dashboard.Version.Status == svcsdktypes.ResourceStatusUpdateSuccessful
	return ready, status
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

// sourceEntityARNsMatch returns true if the desired and latest source entity
// ARNs refer to the same resource. The latest ARN from DescribeDashboard
// includes a /version/N suffix. If the desired ARN is a prefix of the latest
// ARN (i.e. the same base ARN), they match.
func sourceEntityARNsMatch(desired, latest string) bool {
	return strings.HasPrefix(latest, desired)
}

// templateIDFromARN extracts the template ID from a QuickSight template ARN.
// ARN format: arn:aws:quicksight:<region>:<account>:template/<template-id>[/version/<N>]
func templateIDFromARN(arn string) string {
	const prefix = ":template/"
	idx := strings.Index(arn, prefix)
	if idx == -1 {
		return ""
	}
	id := arn[idx+len(prefix):]
	if vIdx := strings.Index(id, "/"); vIdx != -1 {
		id = id[:vIdx]
	}
	return id
}

// resolveDataSetPlaceholders calls DescribeTemplate and returns a map of
// dataset ARN to placeholder name. The DataSetConfigurations in the template
// correspond by position to the DataSetArns on the dashboard version.
func resolveDataSetPlaceholders(
	ctx context.Context,
	sdkapi *svcsdk.Client,
	m *metrics.Metrics,
	awsAccountID *string,
	sourceEntityArn string,
	dataSetArns []string,
) map[string]string {
	result := make(map[string]string, len(dataSetArns))
	templateID := templateIDFromARN(sourceEntityArn)
	if templateID == "" {
		return result
	}
	tplResp, err := sdkapi.DescribeTemplate(ctx, &svcsdk.DescribeTemplateInput{
		AwsAccountId: awsAccountID,
		TemplateId:   &templateID,
	})
	m.RecordAPICall("READ_ONE", "DescribeTemplate", err)
	if err != nil || tplResp.Template == nil || tplResp.Template.Version == nil {
		return result
	}
	configs := tplResp.Template.Version.DataSetConfigurations
	for i, dsARN := range dataSetArns {
		if i < len(configs) && configs[i].Placeholder != nil {
			result[dsARN] = *configs[i].Placeholder
		}
	}
	return result
}
