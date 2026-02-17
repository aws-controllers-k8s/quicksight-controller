# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
# 	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

"""Integration tests for the QuickSight DataSource API.
"""

import pytest
import time
import logging

from acktest.k8s import condition
from acktest.k8s import resource as k8s
from acktest.resources import random_suffix_name
from acktest.aws.identity import get_account_id
from acktest import tags
from acktest.bootstrapping.s3 import Bucket
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e import data_source
from e2e.quicksight_iam import QuickSightTestRole

RESOURCE_PLURAL = "datasources"

CREATE_WAIT_AFTER_SECONDS = 10
UPDATE_WAIT_AFTER_SECONDS = 10
DELETE_WAIT_AFTER_SECONDS = 10


@pytest.fixture
def data_source_s3(request, quicksight_subscription):
    """Fixture to create a DataSource with S3 parameters.
    
    This fixture:
    1. Ensures QuickSight subscription exists (via quicksight_subscription fixture)
    2. Creates a dedicated IAM role for QuickSight to assume when accessing S3
    3. Creates an S3 bucket
    4. Uploads a CSV data file and manifest
    5. Creates a QuickSight DataSource CR
    6. Cleans up all resources after the test (except subscription)
    """
    resource_name = random_suffix_name("qs-datasource", 24)
    data_source_id = resource_name
    aws_account_id = get_account_id()
    
    # Create dedicated test role for QuickSight to assume
    qs_test_role = QuickSightTestRole()
    try:
        qs_test_role.bootstrap()
    except Exception as e:
        pytest.skip(f"Failed to create QuickSight test role: {e}")
    
    # Create S3 bucket using bootstrapping utility
    bucket = Bucket(
        name_prefix="qs-test-bucket"
    )
    bucket.bootstrap()
    
    manifest_key = "manifest.json"
    data_file_key = "data.csv"
    
    try:
        # Create a sample CSV data file
        csv_content = """id,name,value,category
1,Item A,100,electronics
2,Item B,200,books
3,Item C,150,electronics
4,Item D,300,clothing
5,Item E,250,books
"""
        bucket.s3_client.put_object(
            Bucket=bucket.name,
            Key=data_file_key,
            Body=csv_content
        )
        
        # Create manifest file with proper URIs
        manifest_content = f"""{{
  "fileLocations": [
    {{
      "URIs": [
        "s3://{bucket.name}/{data_file_key}"
      ]
    }}
  ],
  "globalUploadSettings": {{
    "format": "CSV",
    "containsHeader": true,
    "delimiter": ","
  }}
}}"""
        bucket.s3_client.put_object(
            Bucket=bucket.name,
            Key=manifest_key,
            Body=manifest_content
        )
        
        logging.info(f"Created S3 bucket {bucket.name} with data file at s3://{bucket.name}/{data_file_key}")
        logging.info(f"Created manifest at s3://{bucket.name}/{manifest_key}")
        logging.info(f"QuickSight will use role: {qs_test_role.role_arn}")
        
    except Exception as e:
        logging.error(f"Failed to create S3 resources: {e}")
        bucket.cleanup()
        qs_test_role.cleanup()
        raise
    
    replacements = REPLACEMENT_VALUES.copy()
    replacements["DATA_SOURCE_NAME"] = resource_name
    replacements["DATA_SOURCE_ID"] = data_source_id
    replacements["AWS_ACCOUNT_ID"] = aws_account_id
    replacements["DATA_SOURCE_TYPE"] = "S3"
    replacements["S3_BUCKET_NAME"] = bucket.name
    replacements["S3_MANIFEST_KEY"] = manifest_key
    replacements["ROLE_ARN"] = qs_test_role.role_arn
    
    # Load DataSource CR
    resource_data = load_resource(
        "data_source",
        additional_replacements=replacements,
    )
    logging.debug(resource_data)
    
    # Create k8s resource
    ref = k8s.CustomResourceReference(
        CRD_GROUP, CRD_VERSION, RESOURCE_PLURAL,
        resource_name, namespace="default",
    )
    k8s.create_custom_resource(ref, resource_data)
    cr = k8s.wait_resource_consumed_by_controller(ref)

    k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
    assert cr is not None
    assert k8s.get_resource_exists(ref)
    
    yield (ref, cr, aws_account_id, data_source_id, bucket.name)
    
    # Cleanup
    if k8s.get_resource_exists(ref):
        _, deleted = k8s.delete_custom_resource(ref)
        assert deleted is True
    
    # Clean up S3 resources using bootstrapping utility
    try:
        bucket.cleanup()
    except Exception as e:
        logging.warning(f"Failed to clean up S3 resources: {e}")
    
    # Clean up QuickSight test role
    try:
        qs_test_role.cleanup()
    except Exception as e:
        logging.warning(f"Failed to clean up QuickSight test role: {e}")


@service_marker
@pytest.mark.canary
class TestDataSource:
    
    def test_basic_data_source(self, quicksight_client, data_source_s3):
        """Test basic DataSource creation and deletion."""
        (ref, cr, aws_account_id, data_source_id, bucket_name) = data_source_s3
        condition.assert_synced(ref)
        
        assert 'spec' in cr
        assert 'dataSourceID' in cr['spec']
        resource_id = cr['spec']['dataSourceID']
        
        # Check QuickSight DataSource exists
        ds = data_source.get_data_source(aws_account_id, resource_id)
        assert ds is not None
        
        # Verify DataSource properties
        assert ds['DataSourceId'] == data_source_id
        assert ds['Type'] == 'S3'
        assert 'DataSourceParameters' in ds
        assert 'S3Parameters' in ds['DataSourceParameters']
        
        # Verify status fields are populated
        assert 'status' in cr
        assert 'ackResourceMetadata' in cr['status']
        assert 'arn' in cr['status']['ackResourceMetadata']
    
    def test_data_source_tags(self, quicksight_client, data_source_s3):
        """Test DataSource tag operations."""
        (ref, cr, aws_account_id, data_source_id, bucket_name) = data_source_s3
        condition.assert_synced(ref)
        
        # Get DataSource ARN
        cr = k8s.get_resource(ref)
        data_source_arn = cr['status']['ackResourceMetadata']['arn']
        
        # Verify initial tags
        ds_tags = data_source.get_tags(aws_account_id, data_source_arn)
        initial_tags = tags.clean(ds_tags)
        assert len(initial_tags) >= 1
        
        # Find the environment tag
        env_tag = next((tag for tag in initial_tags if tag['Key'] == 'environment'), None)
        assert env_tag is not None
        assert env_tag['Value'] == 'test'
        
        # Update tags
        new_tags = [
            {
                "key": "environment",
                "value": "production",
            },
            {
                "key": "team",
                "value": "data-analytics",
            }
        ]
        
        cr = k8s.get_resource(ref)
        cr["spec"]["tags"] = new_tags
        k8s.patch_custom_resource(ref, cr)
        
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Verify updated tags
        ds_tags = data_source.get_tags(aws_account_id, data_source_arn)
        updated_tags = tags.clean(ds_tags)
        
        # Check that both tags are present
        env_tag = next((tag for tag in updated_tags if tag['Key'] == 'environment'), None)
        assert env_tag is not None
        assert env_tag['Value'] == 'production'
        
        team_tag = next((tag for tag in updated_tags if tag['Key'] == 'team'), None)
        assert team_tag is not None
        assert team_tag['Value'] == 'data-analytics'
    
    def test_data_source_update_name_and_role(self, quicksight_client, data_source_s3):
        """Test updating DataSource display name."""
        (ref, cr, aws_account_id, data_source_id, bucket_name) = data_source_s3
        condition.assert_synced(ref)
        
        # Get initial DataSource
        ds = data_source.get_data_source(aws_account_id, data_source_id)
        assert ds is not None
        initial_name = ds['Name']
        
        # Update display name
        new_name = "new_" + initial_name
        cr = k8s.get_resource(ref)
        cr["spec"]["name"] = new_name
        k8s.patch_custom_resource(ref, cr)
        
        time.sleep(UPDATE_WAIT_AFTER_SECONDS)

        k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        
        # Verify name was updated
        ds = data_source.get_data_source(aws_account_id, data_source_id)
        assert ds is not None
        assert ds['Name'] == new_name
        assert ds['Name'] != initial_name

        # Bootstrap a new iam role and update the S3 role arn
        qs_test_role = QuickSightTestRole()


        try:
            qs_test_role.bootstrap()

        except Exception as e:
            pytest.skip(f"Failed to create QuickSight test role: {e}")

        
        # Update role ARN

        cr = k8s.get_resource(ref)
        cr["spec"]["dataSourceParameters"]["s3Parameters"]["roleARN"] = qs_test_role.role_arn
        k8s.patch_custom_resource(ref, cr)

        time.sleep(UPDATE_WAIT_AFTER_SECONDS)
        k8s.wait_on_condition(ref, "ACK.ResourceSynced", "True", wait_periods=5)
        # Verify role ARN was updated
        ds = data_source.get_data_source(aws_account_id, data_source_id)
        assert ds is not None
        assert ds['DataSourceParameters']['S3Parameters']['RoleArn'] == qs_test_role.role_arn

        # cleanup test role
        qs_test_role.cleanup()




