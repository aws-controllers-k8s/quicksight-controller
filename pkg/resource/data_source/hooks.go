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

package data_source

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/quicksight"

	"github.com/aws-controllers-k8s/quicksight-controller/pkg/tags"
)

// resourceTagManager holds methods for working with AWS resource tags
type resourceTagManager struct {
	// tagManager provides methods for working with AWS resource tags
	tagManager *tags.TagManager
	// logConstructor contains a method that can produce a logger for a
	// resource manager from a supplied context.
	logConstructor func(context.Context) tags.ackLogger
}

// newResourceTagManager returns a new resourceTagManager struct
func newResourceTagManager(
	sess *session.Session,
	logConstructor func(context.Context) tags.ackLogger,
) *resourceTagManager {
	return &resourceTagManager{
		tagManager:     tags.NewTagManager(sess, logConstructor),
		logConstructor: logConstructor,
	}
}

// getTags returns the tags for a given resource ARN
func (rtm *resourceTagManager) getTags(
	ctx context.Context,
	resourceARN string,
) []*quicksight.Tag {
	tags, err := rtm.tagManager.GetTags(ctx, resourceARN)
	if err != nil {
		rtm.logConstructor(ctx).Debug("error getting tags for resource", "error", err)
		return nil
	}
	return tags
}

// syncTags synchronizes tags between the supplied desired and latest resources
func (rtm *resourceTagManager) syncTags(
	ctx context.Context,
	latest *resource,
	desired *resource,
) error {
	if latest.Status.ACKResourceMetadata == nil || latest.Status.ACKResourceMetadata.ARN == nil {
		return nil
	}
	resourceARN := *latest.Status.ACKResourceMetadata.ARN

	var latestTags []*quicksight.Tag
	if latest.Spec.Tags != nil {
		latestTags = make([]*quicksight.Tag, len(latest.Spec.Tags))
		for i, tag := range latest.Spec.Tags {
			latestTags[i] = &quicksight.Tag{
				Key:   tag.Key,
				Value: tag.Value,
			}
		}
	}

	var desiredTags []*quicksight.Tag
	if desired.Spec.Tags != nil {
		desiredTags = make([]*quicksight.Tag, len(desired.Spec.Tags))
		for i, tag := range desired.Spec.Tags {
			desiredTags[i] = &quicksight.Tag{
				Key:   tag.Key,
				Value: tag.Value,
			}
		}
	}

	_, err := rtm.tagManager.SyncTags(ctx, resourceARN, desiredTags, latestTags)
	if err != nil {
		return err
	}

	// Update the latest resource's tags to match the desired tags
	latest.Spec.Tags = desired.Spec.Tags

	return nil
}
