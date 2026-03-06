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

package tags_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/quicksight"
	"github.com/aws/aws-sdk-go/service/quicksight/quicksightiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/aws-controllers-k8s/quicksight-controller/pkg/tags"
)

type mockQuickSightClient struct {
	quicksightiface.QuickSightAPI
	mock.Mock
}

func (m *mockQuickSightClient) ListTagsForResourceWithContext(
	ctx context.Context,
	input *quicksight.ListTagsForResourceInput,
	opts ...request.Option,
) (*quicksight.ListTagsForResourceOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*quicksight.ListTagsForResourceOutput), args.Error(1)
}

func (m *mockQuickSightClient) TagResourceWithContext(
	ctx context.Context,
	input *quicksight.TagResourceInput,
	opts ...request.Option,
) (*quicksight.TagResourceOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*quicksight.TagResourceOutput), args.Error(1)
}

func (m *mockQuickSightClient) UntagResourceWithContext(
	ctx context.Context,
	input *quicksight.UntagResourceInput,
	opts ...request.Option,
) (*quicksight.UntagResourceOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*quicksight.UntagResourceOutput), args.Error(1)
}

func TestGetTags(t *testing.T) {
	mockClient := &mockQuickSightClient{}
	tm := tags.NewTagsManager(mockClient)
	ctx := context.Background()
	resourceARN := "arn:aws:quicksight:us-west-2:123456789012:analysis/test-analysis"

	expectedTags := []*quicksight.Tag{
		{
			Key:   aws.String("key1"),
			Value: aws.String("value1"),
		},
		{
			Key:   aws.String("key2"),
			Value: aws.String("value2"),
		},
	}

	mockClient.On("ListTagsForResourceWithContext", ctx, &quicksight.ListTagsForResourceInput{
		ResourceArn: aws.String(resourceARN),
	}).Return(&quicksight.ListTagsForResourceOutput{
		Tags: expectedTags,
	}, nil)

	tags, err := tm.GetTags(ctx, resourceARN)

	assert.NoError(t, err)
	assert.Equal(t, expectedTags, tags)
	mockClient.AssertExpectations(t)
}

func TestSyncTags(t *testing.T) {
	mockClient := &mockQuickSightClient{}
	tm := tags.NewTagsManager(mockClient)
	ctx := context.Background()
	resourceARN := "arn:aws:quicksight:us-west-2:123456789012:analysis/test-analysis"

	// Test adding and removing tags
	desiredTags := []*quicksight.Tag{
		{
			Key:   aws.String("key1"),
			Value: aws.String("new-value1"),
		},
		{
			Key:   aws.String("key3"),
			Value: aws.String("value3"),
		},
	}

	latestTags := []*quicksight.Tag{
		{
			Key:   aws.String("key1"),
			Value: aws.String("value1"),
		},
		{
			Key:   aws.String("key2"),
			Value: aws.String("value2"),
		},
	}

	// Mock UntagResource call for key2
	mockClient.On("UntagResourceWithContext", ctx, &quicksight.UntagResourceInput{
		ResourceArn: aws.String(resourceARN),
		TagKeys:     []*string{aws.String("key2")},
	}).Return(&quicksight.UntagResourceOutput{}, nil)

	// Mock TagResource call for key1 (updated) and key3 (new)
	mockClient.On("TagResourceWithContext", ctx, &quicksight.TagResourceInput{
		ResourceArn: aws.String(resourceARN),
		Tags: []*quicksight.Tag{
			{
				Key:   aws.String("key1"),
				Value: aws.String("new-value1"),
			},
			{
				Key:   aws.String("key3"),
				Value: aws.String("value3"),
			},
		},
	}).Return(&quicksight.TagResourceOutput{}, nil)

	err := tm.SyncTags(ctx, desiredTags, latestTags, resourceARN)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestAddTags(t *testing.T) {
	mockClient := &mockQuickSightClient{}
	tm := tags.NewTagsManager(mockClient)
	ctx := context.Background()
	resourceARN := "arn:aws:quicksight:us-west-2:123456789012:analysis/test-analysis"

	tagsToAdd := []*quicksight.Tag{
		{
			Key:   aws.String("key1"),
			Value: aws.String("value1"),
		},
	}

	mockClient.On("TagResourceWithContext", ctx, &quicksight.TagResourceInput{
		ResourceArn: aws.String(resourceARN),
		Tags:        tagsToAdd,
	}).Return(&quicksight.TagResourceOutput{}, nil)

	err := tm.AddTags(ctx, resourceARN, tagsToAdd)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestRemoveTags(t *testing.T) {
	mockClient := &mockQuickSightClient{}
	tm := tags.NewTagsManager(mockClient)
	ctx := context.Background()
	resourceARN := "arn:aws:quicksight:us-west-2:123456789012:analysis/test-analysis"

	tagsToRemove := []*quicksight.Tag{
		{
			Key:   aws.String("key1"),
			Value: aws.String("value1"),
		},
	}

	mockClient.On("UntagResourceWithContext", ctx, &quicksight.UntagResourceInput{
		ResourceArn: aws.String(resourceARN),
		TagKeys:     []*string{aws.String("key1")},
	}).Return(&quicksight.UntagResourceOutput{}, nil)

	err := tm.RemoveTags(ctx, resourceARN, tagsToRemove)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}
