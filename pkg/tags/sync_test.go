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
	"testing"

	"github.com/aws-controllers-k8s/quicksight-controller/pkg/tags"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/quicksight"
	"github.com/stretchr/testify/assert"
)

func TestTagsChanged(t *testing.T) {
	tests := []struct {
		name        string
		desiredTags []*quicksight.Tag
		latestTags  []*quicksight.Tag
		expected    bool
	}{
		{
			name: "identical tags",
			desiredTags: []*quicksight.Tag{
				{
					Key:   aws.String("key1"),
					Value: aws.String("value1"),
				},
				{
					Key:   aws.String("key2"),
					Value: aws.String("value2"),
				},
			},
			latestTags: []*quicksight.Tag{
				{
					Key:   aws.String("key1"),
					Value: aws.String("value1"),
				},
				{
					Key:   aws.String("key2"),
					Value: aws.String("value2"),
				},
			},
			expected: false,
		},
		{
			name: "different tag values",
			desiredTags: []*quicksight.Tag{
				{
					Key:   aws.String("key1"),
					Value: aws.String("value1"),
				},
				{
					Key:   aws.String("key2"),
					Value: aws.String("newvalue2"),
				},
			},
			latestTags: []*quicksight.Tag{
				{
					Key:   aws.String("key1"),
					Value: aws.String("value1"),
				},
				{
					Key:   aws.String("key2"),
					Value: aws.String("value2"),
				},
			},
			expected: true,
		},
		{
			name: "different tag keys",
			desiredTags: []*quicksight.Tag{
				{
					Key:   aws.String("key1"),
					Value: aws.String("value1"),
				},
				{
					Key:   aws.String("key3"),
					Value: aws.String("value3"),
				},
			},
			latestTags: []*quicksight.Tag{
				{
					Key:   aws.String("key1"),
					Value: aws.String("value1"),
				},
				{
					Key:   aws.String("key2"),
					Value: aws.String("value2"),
				},
			},
			expected: true,
		},
		{
			name: "different number of tags",
			desiredTags: []*quicksight.Tag{
				{
					Key:   aws.String("key1"),
					Value: aws.String("value1"),
				},
				{
					Key:   aws.String("key2"),
					Value: aws.String("value2"),
				},
				{
					Key:   aws.String("key3"),
					Value: aws.String("value3"),
				},
			},
			latestTags: []*quicksight.Tag{
				{
					Key:   aws.String("key1"),
					Value: aws.String("value1"),
				},
				{
					Key:   aws.String("key2"),
					Value: aws.String("value2"),
				},
			},
			expected: true,
		},
		{
			name:        "empty desired tags",
			desiredTags: []*quicksight.Tag{},
			latestTags: []*quicksight.Tag{
				{
					Key:   aws.String("key1"),
					Value: aws.String("value1"),
				},
			},
			expected: true,
		},
		{
			name: "empty latest tags",
			desiredTags: []*quicksight.Tag{
				{
					Key:   aws.String("key1"),
					Value: aws.String("value1"),
				},
			},
			latestTags: []*quicksight.Tag{},
			expected:   true,
		},
		{
			name:        "both empty",
			desiredTags: []*quicksight.Tag{},
			latestTags:  []*quicksight.Tag{},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tags.TagsChanged(tt.desiredTags, tt.latestTags)
			assert.Equal(t, tt.expected, result)
		})
	}
}
