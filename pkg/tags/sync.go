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

package tags

import (
	"context"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/quicksight"
	"github.com/aws/aws-sdk-go/service/quicksight/quicksightiface"
)

// TagType represents the AWS QuickSight Tag structure
type TagType struct {
	Key   *string
	Value *string
}

// TagManager provides methods for working with AWS QuickSight tags
type TagManager struct {
	client quicksightiface.QuickSightAPI
	// logConstructor contains a method that can produce a logger for a
	// resource manager
	logConstructor func(o ...ackrtlog.Option) *ackrtlog.Logger
}

// NewTagManager creates a new TagManager instance
func NewTagManager(
	client quicksightiface.QuickSightAPI,
	logConstructor func(o ...ackrtlog.Option) *ackrtlog.Logger,
) *TagManager {
	return &TagManager{
		client:         client,
		logConstructor: logConstructor,
	}
}

// GetTags returns the tags for a given resource ARN
func (tm *TagManager) GetTags(
	ctx context.Context,
	resourceARN string,
) ([]*TagType, error) {
	logger := tm.logConstructor()
	logger.Debug("getting tags for resource", "resource_arn", resourceARN)

	input := &quicksight.ListTagsForResourceInput{
		ResourceArn: aws.String(resourceARN),
	}

	resp, err := tm.client.ListTagsForResourceWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	tags := make([]*TagType, len(resp.Tags))
	for i, tag := range resp.Tags {
		tags[i] = &TagType{
			Key:   tag.Key,
			Value: tag.Value,
		}
	}

	return tags, nil
}

// SyncTags synchronizes the tags between the resource spec and the AWS resource
func (tm *TagManager) SyncTags(
	ctx context.Context,
	resourceARN string,
	desiredTags []*TagType,
	latestTags []*TagType,
) (bool, error) {
	logger := tm.logConstructor()
	logger.Debug("syncing tags for resource", "resource_arn", resourceARN)

	// If there are no desired tags and no latest tags, nothing to do
	if len(desiredTags) == 0 && len(latestTags) == 0 {
		return false, nil
	}

	// If there are no desired tags but there are latest tags, remove all tags
	if len(desiredTags) == 0 && len(latestTags) > 0 {
		tagKeys := make([]*string, len(latestTags))
		for i, tag := range latestTags {
			tagKeys[i] = tag.Key
		}

		_, err := tm.client.UntagResourceWithContext(
			ctx,
			&quicksight.UntagResourceInput{
				ResourceArn: aws.String(resourceARN),
				TagKeys:     tagKeys,
			},
		)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	// Convert the tags to maps for easier comparison
	desiredTagMap := make(map[string]string)
	for _, tag := range desiredTags {
		if tag.Key != nil && tag.Value != nil {
			desiredTagMap[*tag.Key] = *tag.Value
		}
	}

	latestTagMap := make(map[string]string)
	for _, tag := range latestTags {
		if tag.Key != nil && tag.Value != nil {
			latestTagMap[*tag.Key] = *tag.Value
		}
	}

	// Determine which tags to add or update
	tagsToAddOrUpdate := []*quicksight.Tag{}
	for key, value := range desiredTagMap {
		latestValue, exists := latestTagMap[key]
		if !exists || value != latestValue {
			tagsToAddOrUpdate = append(tagsToAddOrUpdate, &quicksight.Tag{
				Key:   aws.String(key),
				Value: aws.String(value),
			})
		}
	}

	// Determine which tags to remove
	tagsToRemove := []*string{}
	for key := range latestTagMap {
		_, exists := desiredTagMap[key]
		if !exists {
			tagsToRemove = append(tagsToRemove, aws.String(key))
		}
	}

	changed := false

	// Add or update tags if needed
	if len(tagsToAddOrUpdate) > 0 {
		_, err := tm.client.TagResourceWithContext(
			ctx,
			&quicksight.TagResourceInput{
				ResourceArn: aws.String(resourceARN),
				Tags:        tagsToAddOrUpdate,
			},
		)
		if err != nil {
			return false, err
		}
		changed = true
	}

	// Remove tags if needed
	if len(tagsToRemove) > 0 {
		_, err := tm.client.UntagResourceWithContext(
			ctx,
			&quicksight.UntagResourceInput{
				ResourceArn: aws.String(resourceARN),
				TagKeys:     tagsToRemove,
			},
		)
		if err != nil {
			return false, err
		}
		changed = true
	}

	return changed, nil
}

// ConvertTagsToACK converts AWS QuickSight tags to ACK tags
func ConvertTagsToACK(tags []*TagType) []*ackcompare.Tag {
	if len(tags) == 0 {
		return nil
	}

	res := make([]*ackcompare.Tag, len(tags))
	for i, tag := range tags {
		res[i] = &ackcompare.Tag{
			Key:   *tag.Key,
			Value: *tag.Value,
		}
	}
	return res
}

// ConvertACKTags converts ACK tags to AWS QuickSight tags
func ConvertACKTags(tags []*ackcompare.Tag) []*TagType {
	if len(tags) == 0 {
		return nil
	}

	res := make([]*TagType, len(tags))
	for i, tag := range tags {
		res[i] = &TagType{
			Key:   aws.String(tag.Key),
			Value: aws.String(tag.Value),
		}
	}
	return res
}
