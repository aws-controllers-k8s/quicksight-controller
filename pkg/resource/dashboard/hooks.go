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

	"github.com/aws-controllers-k8s/quicksight-controller/pkg/tags"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	"github.com/aws/aws-sdk-go/service/quicksight"
)

// resourceManager is responsible for providing a consistent way to perform
// CRUD operations in a backend AWS service API for Dashboard resources.
type resourceManager struct {
	// Client represents an AWS service API client
	sdkapi quicksight.QuickSightAPI
	// tagManager provides methods for working with AWS QuickSight tags
	tagManager *tags.TagManager
}

// getTags returns the tags for a given Dashboard resource
func (rm *resourceManager) getTags(
	ctx context.Context,
	resourceARN string,
) []*ackcompare.Tag {
	if resourceARN == "" {
		return nil
	}

	tagList, err := rm.tagManager.GetTags(ctx, resourceARN)
	if err != nil {
		return nil
	}

	return tags.ConvertTagsToACK(tagList)
}

// syncTags synchronizes the tags between the Dashboard resource spec and the AWS resource
func (rm *resourceManager) syncTags(
	ctx context.Context,
	latest *resource,
	desired *resource,
) error {
	if latest.ko.Status.ACKResourceMetadata == nil || latest.ko.Status.ACKResourceMetadata.ARN == nil {
		return nil
	}

	resourceARN := *latest.ko.Status.ACKResourceMetadata.ARN

	var latestTags []*tags.TagType
	if latest.ko.Spec.Tags != nil {
		latestACKTags := make([]*ackcompare.Tag, len(latest.ko.Spec.Tags))
		for i, tag := range latest.ko.Spec.Tags {
			latestACKTags[i] = &ackcompare.Tag{
				Key:   *tag.Key,
				Value: *tag.Value,
			}
		}
		latestTags = tags.ConvertACKTags(latestACKTags)
	}

	var desiredTags []*tags.TagType
	if desired.ko.Spec.Tags != nil {
		desiredACKTags := make([]*ackcompare.Tag, len(desired.ko.Spec.Tags))
		for i, tag := range desired.ko.Spec.Tags {
			desiredACKTags[i] = &ackcompare.Tag{
				Key:   *tag.Key,
				Value: *tag.Value,
			}
		}
		desiredTags = tags.ConvertACKTags(desiredACKTags)
	}

	_, err := rm.tagManager.SyncTags(ctx, resourceARN, desiredTags, latestTags)
	return err
}

// newTagManager returns a new tagManager for the Dashboard resource
func (rm *resourceManager) newTagManager() {
	rm.tagManager = tags.NewTagManager(rm.sdkapi, rm.log)
}
