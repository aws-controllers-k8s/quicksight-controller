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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/quicksight"
	"github.com/aws/aws-sdk-go/service/quicksight/quicksightiface"
)

// TagManager provides methods for working with AWS resource tags
type TagManager struct {
	client quicksightiface.QuickSightAPI
	// logConstructor contains a method that can produce a logger for a
	// resource manager from a supplied context.
	logConstructor func(context.Context) ackLogger
}

// ackLogger is a minimal interface for loggers used in the ACK runtime
type ackLogger interface {
	Debug(msg string, additionalFields ...interface{})
}

// NewTagManager creates a new TagManager instance
func NewTagManager(
	sess *session.Session,
	logConstructor func(context.Context) ackLogger,
) *TagManager {
	return &TagManager{
		client:         quicksight.New(sess),
		logConstructor: logConstructor,
	}
}

// GetTags returns the tags for a given resource ARN
func (tm *TagManager) GetTags(
	ctx context.Context,
	resourceARN string,
) ([]*quicksight.Tag, error) {
	logger := tm.logConstructor(ctx)
	logger.Debug("getting tags for resource", "resource_arn", resourceARN)

	input := &quicksight.ListTagsForResourceInput{
		ResourceArn: aws.String(resourceARN),
	}

	resp, err := tm.client.ListTagsForResourceWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	return resp.Tags, nil
}

// SyncTags synchronizes tags between the supplied desired and latest resources
func (tm *TagManager) SyncTags(
	ctx context.Context,
	resourceARN string,
	desiredTags []*quicksight.Tag,
	latestTags []*quicksight.Tag,
) (bool, error) {
	logger := tm.logConstructor(ctx)
	logger.Debug("syncing tags for resource", "resource_arn", resourceARN)

	// If there are no differences, return early
	if !TagsChanged(desiredTags, latestTags) {
		return false, nil
	}

	// Determine which tags to add and which to remove
	tagsToAdd := []*quicksight.Tag{}
	tagKeysToRemove := []*string{}

	// Build a map of existing tags for easier lookup
	existingTagMap := make(map[string]string)
	for _, tag := range latestTags {
		existingTagMap[*tag.Key] = *tag.Value
	}

	// Build a map of desired tags for easier lookup
	desiredTagMap := make(map[string]string)
	for _, tag := range desiredTags {
		desiredTagMap[*tag.Key] = *tag.Value
	}

	// Find tags to add or update
	for _, tag := range desiredTags {
		key := *tag.Key
		value := *tag.Value
		existingValue, exists := existingTagMap[key]
		if !exists || existingValue != value {
			tagsToAdd = append(tagsToAdd, tag)
		}
	}

	// Find tags to remove
	for _, tag := range latestTags {
		key := *tag.Key
		_, exists := desiredTagMap[key]
		if !exists {
			tagKeysToRemove = append(tagKeysToRemove, tag.Key)
		}
	}

	// Add new or updated tags
	if len(tagsToAdd) > 0 {
		_, err := tm.client.TagResourceWithContext(
			ctx,
			&quicksight.TagResourceInput{
				ResourceArn: aws.String(resourceARN),
				Tags:        tagsToAdd,
			},
		)
		if err != nil {
			return false, err
		}
	}

	// Remove tags
	if len(tagKeysToRemove) > 0 {
		_, err := tm.client.UntagResourceWithContext(
			ctx,
			&quicksight.UntagResourceInput{
				ResourceArn: aws.String(resourceARN),
				TagKeys:     tagKeysToRemove,
			},
		)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

// TagsChanged returns true if there are differences between two tag lists
func TagsChanged(
	desiredTags []*quicksight.Tag,
	latestTags []*quicksight.Tag,
) bool {
	if len(desiredTags) != len(latestTags) {
		return true
	}

	// Build maps for easier comparison
	desiredTagMap := make(map[string]string)
	for _, tag := range desiredTags {
		desiredTagMap[*tag.Key] = *tag.Value
	}

	latestTagMap := make(map[string]string)
	for _, tag := range latestTags {
		latestTagMap[*tag.Key] = *tag.Value
	}

	// Check if all desired tags exist with the same values
	for key, desiredValue := range desiredTagMap {
		latestValue, exists := latestTagMap[key]
		if !exists || latestValue != desiredValue {
			return true
		}
	}

	// Check if all latest tags exist in desired tags
	for key := range latestTagMap {
		_, exists := desiredTagMap[key]
		if !exists {
			return true
		}
	}

	return false
}
