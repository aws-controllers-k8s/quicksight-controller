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

	"github.com/aws-controllers-k8s/quicksight-controller/pkg/tags"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/quicksight"
	"github.com/aws/aws-sdk-go/service/quicksight/quicksightiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockQuickSightClient struct {
	quicksightiface.QuickSightAPI
	mock.Mock
}

func (m *mockQuickSightClient) ListTagsForResourceWithContext(ctx context.Context, input *quicksight.ListTagsForResourceInput, opts ...interface{}) (*quicksight.ListTagsForResourceOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*quicksight.ListTagsForResourceOutput), args.Error(1)
}

func (m *mockQuickSightClient) TagResourceWithContext(ctx context.Context, input *quicksight.TagResourceInput, opts ...interface{}) (*quicksight.TagResourceOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*quicksight.TagResourceOutput), args.Error(1)
}

func (m *mockQuickSightClient) UntagResourceWithContext(ctx context.Context, input *quicksight.UntagResourceInput, opts ...interface{}) (*quicksight.UntagResourceOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*quicksight.UntagResourceOutput), args.Error(1)
}

func TestGetTags(t *testing.T) {
	mockClient := new(mockQuickSightClient)
	resourceARN := "arn:aws:quicksight:us-west-2:123456789012:dashboard/test-dashboard"

	mockClient.On("ListTagsForResourceWithContext", mock.Anything, &quicksight.ListTagsForResourceInput{
		ResourceArn: aws.String(resourceARN),
	}).Return(&quicksight.ListTagsForResourceOutput{
		Tags: []*quicksight.Tag{
			{
				Key:   aws.String("key1"),
				Value: aws.String("value1"),
			},
			{
				Key:   aws.String("key2"),
				Value: aws.String("value2"),
			},
		},
	}, nil)

	logConstructor := func(o ...ackrtlog.Option) *ackrtlog.Logger {
		return ackrtlog.New(o...)
	}

	tagManager := tags.NewTagManager(mockClient, logConstructor)
	result, err := tagManager.GetTags(context.Background(), resourceARN)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "key1", *result[0].Key)
	assert.Equal(t, "value1", *result[0].Value)
	assert.Equal(t, "key2", *result[1].Key)
	assert.Equal(t, "value2", *result[1].Value)

	mockClient.AssertExpectations(t)
}

func TestSyncTags_AddAndRemove(t *testing.T) {
	mockClient := new(mockQuickSightClient)
	resourceARN := "arn:aws:quicksight:us-west-2:123456789012:dashboard/test-dashboard"

	// Set up desired and latest tags
	desiredTags := []*tags.TagType{
		{
			Key:   aws.String("key1"),
			Value: aws.String("newvalue1"),
		},
		{
			Key:   aws.String("key3"),
			Value: aws.String("value3"),
		},
	}

	latestTags := []*tags.TagType{
		{
			Key:   aws.String("key1"),
			Value: aws.String("value1"),
		},
		{
			Key:   aws.String("key2"),
			Value: aws.String("value2"),
		},
	}

	// Mock TagResource call
	mockClient.On("TagResourceWithContext", mock.Anything, &quicksight.TagResourceInput{
		ResourceArn: aws.String(resourceARN),
		Tags: []*quicksight.Tag{
			{
				Key:   aws.String("key1"),
				Value: aws.String("newvalue1"),
			},
			{
				Key:   aws.String("key3"),
				Value: aws.String("value3"),
			},
		},
	}).Return(&quicksight.TagResourceOutput{}, nil)

	// Mock UntagResource call
	mockClient.On("UntagResourceWithContext", mock.Anything, &quicksight.UntagResourceInput{
		ResourceArn: aws.String(resourceARN),
		TagKeys:     []*string{aws.String("key2")},
	}).Return(&quicksight.UntagResourceOutput{}, nil)

	logConstructor := func(o ...ackrtlog.Option) *ackrtlog.Logger {
		return ackrtlog.New(o...)
	}

	tagManager := tags.NewTagManager(mockClient, logConstructor)
	changed, err := tagManager.SyncTags(context.Background(), resourceARN, desiredTags, latestTags)

	assert.NoError(t, err)
	assert.True(t, changed)
	mockClient.AssertExpectations(t)
}

func TestSyncTags_NoChanges(t *testing.T) {
	mockClient := new(mockQuickSightClient)
	resourceARN := "arn:aws:quicksight:us-west-2:123456789012:dashboard/test-dashboard"

	// Set up identical desired and latest tags
	desiredTags := []*tags.TagType{
		{
			Key:   aws.String("key1"),
			Value: aws.String("value1"),
		},
		{
			Key:   aws.String("key2"),
			Value: aws.String("value2"),
		},
	}

	latestTags := []*tags.TagType{
		{
			Key:   aws.String("key1"),
			Value: aws.String("value1"),
		},
		{
			Key:   aws.String("key2"),
			Value: aws.String("value2"),
		},
	}

	logConstructor := func(o ...ackrtlog.Option) *ackrtlog.Logger {
		return ackrtlog.New(o...)
	}

	tagManager := tags.NewTagManager(mockClient, logConstructor)
	changed, err := tagManager.SyncTags(context.Background(), resourceARN, desiredTags, latestTags)

	assert.NoError(t, err)
	assert.False(t, changed)
	// No API calls should be made when tags are identical
	mockClient.AssertNotCalled(t, "TagResourceWithContext")
	mockClient.AssertNotCalled(t, "UntagResourceWithContext")
}

func TestConvertTagsToACK(t *testing.T) {
	inputTags := []*tags.TagType{
		{
			Key:   aws.String("key1"),
			Value: aws.String("value1"),
		},
		{
			Key:   aws.String("key2"),
			Value: aws.String("value2"),
		},
	}

	result := tags.ConvertTagsToACK(inputTags)

	assert.Len(t, result, 2)
	assert.Equal(t, "key1", result[0].Key)
	assert.Equal(t, "value1", result[0].Value)
	assert.Equal(t, "key2", result[1].Key)
	assert.Equal(t, "value2", result[1].Value)
}

func TestConvertACKTags(t *testing.T) {
	inputTags := []*ackcompare.Tag{
		{
			Key:   "key1",
			Value: "value1",
		},
		{
			Key:   "key2",
			Value: "value2",
		},
	}

	result := tags.ConvertACKTags(inputTags)

	assert.Len(t, result, 2)
	assert.Equal(t, "key1", *result[0].Key)
	assert.Equal(t, "value1", *result[0].Value)
	assert.Equal(t, "key2", *result[1].Key)
	assert.Equal(t, "value2", *result[1].Value)
}
