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
	acktags "github.com/aws-controllers-k8s/runtime/pkg/tags"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/quicksight"
	"github.com/aws/aws-sdk-go/service/quicksight/quicksightiface"
)

// TagsManager provides methods for working with QuickSight tags
type TagsManager struct {
	client quicksightiface.QuickSightAPI
}

// NewTagsManager creates a new TagsManager
func NewTagsManager(client quicksightiface.QuickSightAPI) *TagsManager {
	return &TagsManager{
		client: client,
	}
}

// GetTags returns the tags for a given resource ARN
func (tm *TagsManager) GetTags(
	ctx context.Context,
	resourceARN string,
) ([]*quicksight.Tag, error) {
	input := &quicksight.ListTagsForResourceInput{
		ResourceArn: aws.String(resourceARN),
	}

	req := tm.client.ListTagsForResourceWithContext(ctx, input)
	req.SetContext(ctx)

	resp, err := req.Send()
	if err != nil {
		return nil, err
	}

	return resp.Tags, nil
}

// SyncTags synchronizes the tags between the ACK resource and the AWS resource
func (tm *TagsManager) SyncTags(
	ctx context.Context,
	desired []*quicksight.Tag,
	latest []*quicksight.Tag,
	resourceARN string,
) error {
	added, removed := acktags.DiffTagSets(desired, latest, tagKeyValueToACKTag, ackTagToTagKeyValue)

	if len(removed) > 0 {
		if err := tm.RemoveTags(ctx, resourceARN, removed); err != nil {
			return err
		}
	}

	if len(added) > 0 {
		if err := tm.AddTags(ctx, resourceARN, added); err != nil {
			return err
		}
	}

	return nil
}

// AddTags adds tags to a resource
func (tm *TagsManager) AddTags(
	ctx context.Context,
	resourceARN string,
	tags []*quicksight.Tag,
) error {
	input := &quicksight.TagResourceInput{
		ResourceArn: aws.String(resourceARN),
		Tags:        tags,
	}

	req := tm.client.TagResourceWithContext(ctx, input)
	req.SetContext(ctx)

	_, err := req.Send()
	if err != nil {
		return err
	}

	return nil
}

// RemoveTags removes tags from a resource
func (tm *TagsManager) RemoveTags(
	ctx context.Context,
	resourceARN string,
	tags []*quicksight.Tag,
) error {
	tagKeys := make([]*string, 0, len(tags))
	for _, tag := range tags {
		tagKeys = append(tagKeys, tag.Key)
	}

	input := &quicksight.UntagResourceInput{
		ResourceArn: aws.String(resourceARN),
		TagKeys:     tagKeys,
	}

	req := tm.client.UntagResourceWithContext(ctx, input)
	req.SetContext(ctx)

	_, err := req.Send()
	if err != nil {
		return err
	}

	return nil
}

// tagKeyValueToACKTag converts a QuickSight Tag to an ACK Tag
func tagKeyValueToACKTag(
	tag *quicksight.Tag,
) ackcompare.Tag {
	return ackcompare.Tag{
		Key:   *tag.Key,
		Value: *tag.Value,
	}
}

// ackTagToTagKeyValue converts an ACK Tag to a QuickSight Tag
func ackTagToTagKeyValue(
	tag ackcompare.Tag,
) *quicksight.Tag {
	return &quicksight.Tag{
		Key:   aws.String(tag.Key),
		Value: aws.String(tag.Value),
	}
}
