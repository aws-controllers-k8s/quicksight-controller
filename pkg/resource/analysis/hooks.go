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

package analysis

import (
	"context"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	acktags "github.com/aws-controllers-k8s/runtime/pkg/tags"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/quicksight"
	corev1 "k8s.io/api/core/v1"
)

// customTagsManager provides methods for working with QuickSight tags
type customTagsManager struct {
	rm *resourceManager
}

// newCustomTagsManager creates a new customTagsManager
func newCustomTagsManager(rm *resourceManager) *customTagsManager {
	return &customTagsManager{
		rm: rm,
	}
}

// getTags returns the tags for a given Analysis resource
func (rm *resourceManager) getTags(
	ctx context.Context,
	resourceARN string,
) []*quicksight.Tag {
	input := &quicksight.ListTagsForResourceInput{
		ResourceArn: aws.String(resourceARN),
	}

	resp, err := rm.sdkapi.ListTagsForResourceWithContext(ctx, input)
	if err != nil {
		return nil
	}

	return resp.Tags
}

// syncTags synchronizes tags between the ACK resource and the AWS resource
func (rm *resourceManager) syncTags(
	ctx context.Context,
	latest *resource,
	desired *resource,
) error {
	if latest.ko.Status.ACKResourceMetadata == nil || latest.ko.Status.ACKResourceMetadata.ARN == nil {
		return nil
	}

	resourceARN := *latest.ko.Status.ACKResourceMetadata.ARN

	var latestTags []*quicksight.Tag
	if latest.ko.Spec.Tags != nil {
		latestTags = latest.ko.Spec.Tags
	}

	var desiredTags []*quicksight.Tag
	if desired.ko.Spec.Tags != nil {
		desiredTags = desired.ko.Spec.Tags
	}

	added, removed := acktags.DiffTagSets(desiredTags, latestTags, tagKeyValueToACKTag, ackTagToTagKeyValue)

	if len(removed) > 0 {
		if err := rm.removeTags(ctx, resourceARN, removed); err != nil {
			return err
		}
	}

	if len(added) > 0 {
		if err := rm.addTags(ctx, resourceARN, added); err != nil {
			return err
		}
	}

	return nil
}

// addTags adds tags to a resource
func (rm *resourceManager) addTags(
	ctx context.Context,
	resourceARN string,
	tags []*quicksight.Tag,
) error {
	input := &quicksight.TagResourceInput{
		ResourceArn: aws.String(resourceARN),
		Tags:        tags,
	}

	_, err := rm.sdkapi.TagResourceWithContext(ctx, input)
	if err != nil {
		return err
	}

	return nil
}

// removeTags removes tags from a resource
func (rm *resourceManager) removeTags(
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

	_, err := rm.sdkapi.UntagResourceWithContext(ctx, input)
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

// CustomCreateAnalysisPostSetOutput is a custom hook for the CreateAnalysis operation
func (rm *resourceManager) CustomCreateAnalysisPostSetOutput(
	ctx context.Context,
	r *resource,
) error {
	if r.ko.Spec.Tags != nil {
		ackcondition.SetSynced(r, corev1.ConditionFalse, nil, nil)
	}
	return nil
}

// CustomUpdateAnalysisPreBuildRequest is a custom hook for the UpdateAnalysis operation
func (rm *resourceManager) CustomUpdateAnalysisPreBuildRequest(
	ctx context.Context,
	latest *resource,
	desired *resource,
	delta *ackcompare.Delta,
) (*resource, error) {
	if delta.DifferentAt("Spec.Tags") {
		err := rm.syncTags(
			ctx,
			latest,
			desired,
		)
		if err != nil {
			return nil, err
		}
	}
	if !delta.DifferentExcept("Spec.Tags") {
		return desired, nil
	}
	return desired, nil
}

// CustomReadAnalysisPostSetOutput is a custom hook for the ReadAnalysis operation
func (rm *resourceManager) CustomReadAnalysisPostSetOutput(
	ctx context.Context,
	r *resource,
) error {
	if r.ko.Status.ACKResourceMetadata != nil && r.ko.Status.ACKResourceMetadata.ARN != nil {
		r.ko.Spec.Tags = rm.getTags(ctx, *r.ko.Status.ACKResourceMetadata.ARN)
	}
	return nil
}
