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

"""Integration tests for QuickSight Dashboard adoption behavior.

Validates that when a dashboard with multiple published revisions is adopted
by ACK, the controller preserves the current published version and does not
publish a newer draft version.
"""

import pytest
import time
import logging

from acktest.resources import random_suffix_name
from acktest.k8s import resource as k8s
from acktest.aws.identity import get_account_id
from e2e import service_marker, CRD_GROUP, CRD_VERSION, load_resource
from e2e.replacement_values import REPLACEMENT_VALUES
from e2e.bootstrap_resources import get_bootstrap_resources
from e2e.tests.test_dashboard import (
    _create_data_source,
    _create_data_set,
    _create_template,
    _delete_template,
    DASHBOARD_RESOURCE_PLURAL,
    MODIFY_WAIT_AFTER_SECONDS,
    TEMPLATE_WAIT_SECONDS,
    DASHBOARD_SYNC_WAIT_PERIODS,
)

DASHBOARD_CREATION_WAIT_SECONDS = 30


def _wait_dashboard_version_status(
    quicksight_client, aws_account_id, dashboard_id, version_number, target_statuses,
    max_attempts=20, wait_seconds=5,
):
    """Poll DescribeDashboard for a specific version until it reaches one of
    the target statuses. Returns the version status string.
    """
    for _ in range(max_attempts):
        resp = quicksight_client.describe_dashboard(
            AwsAccountId=aws_account_id,
            DashboardId=dashboard_id,
            VersionNumber=version_number,
        )
        status = resp["Dashboard"]["Version"]["Status"]
        if status in target_statuses:
            return status
        time.sleep(wait_seconds)
    raise Exception(
        f"Dashboard {dashboard_id} version {version_number} did not reach "
        f"{target_statuses} in time; last status: {status}"
    )


def _create_dashboard_via_boto(
    quicksight_client, dashboard_id, dashboard_name, template_arn,
    data_set_arn, aws_account_id,
):
    """Create a dashboard directly via boto3 (not through ACK).
    Returns the dashboard ARN.
    """
    resp = quicksight_client.create_dashboard(
        AwsAccountId=aws_account_id,
        DashboardId=dashboard_id,
        Name=dashboard_name,
        SourceEntity={
            "SourceTemplate": {
                "Arn": template_arn,
                "DataSetReferences": [
                    {
                        "DataSetArn": data_set_arn,
                        "DataSetPlaceholder": "testDataSet",
                    },
                ],
            },
        },
    )
    return resp["Arn"]


def _update_dashboard_via_boto(
    quicksight_client, dashboard_id, dashboard_name, template_arn,
    data_set_arn, aws_account_id,
):
    """Update a dashboard directly via boto3 to create a new draft version.
    Returns the version ARN.
    """
    resp = quicksight_client.update_dashboard(
        AwsAccountId=aws_account_id,
        DashboardId=dashboard_id,
        Name=dashboard_name,
        SourceEntity={
            "SourceTemplate": {
                "Arn": template_arn,
                "DataSetReferences": [
                    {
                        "DataSetArn": data_set_arn,
                        "DataSetPlaceholder": "testDataSet",
                    },
                ],
            },
        },
    )
    return resp.get("VersionArn")


def _delete_dashboard_via_boto(quicksight_client, dashboard_id, aws_account_id):
    """Delete a dashboard directly via boto3."""
    try:
        quicksight_client.delete_dashboard(
            AwsAccountId=aws_account_id,
            DashboardId=dashboard_id,
        )
    except Exception:
        logging.warning(f"Failed to delete dashboard {dashboard_id}", exc_info=True)


@pytest.fixture(scope="module")
def adoption_dependencies(quicksight_client):
    """Creates DataSource, DataSet, and Template as dependencies for adoption tests.
    Yields (data_source_ref, data_set_ref, template_arn, aws_account_id, data_set_arn).
    """
    ds_name = random_suffix_name("ack-test-ds-adopt", 32)
    (ds_ref, ds_cr, aws_account_id) = _create_data_source(ds_name)
    data_source_arn = ds_cr["status"]["ackResourceMetadata"]["arn"]
    logging.info(f"Created DataSource {ds_name} with ARN {data_source_arn}")

    dset_name = random_suffix_name("ack-test-dset-adopt", 32)
    (dset_ref, dset_cr, data_set_arn) = _create_data_set(
        dset_name, data_source_arn, aws_account_id,
    )
    logging.info(f"Created DataSet {dset_name} with ARN {data_set_arn}")

    template_id = random_suffix_name("ack-test-tpl-adopt", 32)
    template_arn = _create_template(
        quicksight_client, template_id, template_id,
        data_set_arn, aws_account_id,
    )
    logging.info(f"Created Template {template_id} with ARN {template_arn}")

    yield (ds_ref, dset_ref, template_arn, aws_account_id, data_set_arn)

    # Teardown
    _delete_template(quicksight_client, template_id, aws_account_id)
    try:
        _, deleted = k8s.delete_custom_resource(dset_ref, 3, 10)
        assert deleted
    except:
        pass
    time.sleep(MODIFY_WAIT_AFTER_SECONDS)
    try:
        _, deleted = k8s.delete_custom_resource(ds_ref, 3, 10)
        assert deleted
    except:
        pass


@service_marker
@pytest.mark.canary
class TestDashboardAdoption:
    def test_adopt_preserves_published_version(
        self, quicksight_client, adoption_dependencies,
    ):
        """When a dashboard has multiple versions but is published on an older
        revision, adopting it into ACK should preserve the published version's
        versionNumber and versionStatus. The controller must not publish the
        newer draft version.

        Steps:
        1. Create dashboard via boto3 (version 1, auto-published)
        2. Wait for version 1 to reach CREATION_SUCCESSFUL
        3. Update dashboard via boto3 (creates version 2 as draft, not published)
        4. Wait for version 2 to reach CREATION_SUCCESSFUL
        5. Adopt the dashboard into ACK
        6. Verify versionNumber == 1 and versionStatus == CREATION_SUCCESSFUL
        7. Verify AWS still has version 1 as the published version
        """
        (_, _, template_arn, aws_account_id, data_set_arn) = adoption_dependencies
        dashboard_id = random_suffix_name("ack-adopt-dash", 24)
        dashboard_name = dashboard_id

        # Step 1: Create dashboard via boto3 (version 1 auto-published)
        _create_dashboard_via_boto(
            quicksight_client, dashboard_id, dashboard_name,
            template_arn, data_set_arn, aws_account_id,
        )
        logging.info(f"Created dashboard {dashboard_id} via boto3")

        # Step 2: Wait for version 1 to be ready
        v1_status = _wait_dashboard_version_status(
            quicksight_client, aws_account_id, dashboard_id,
            version_number=1,
            target_statuses=["CREATION_SUCCESSFUL"],
        )
        logging.info(f"Dashboard version 1 status: {v1_status}")

        # Step 3: Update dashboard to create version 2 (draft, not published)
        _update_dashboard_via_boto(
            quicksight_client, dashboard_id, f"{dashboard_name}-v2",
            template_arn, data_set_arn, aws_account_id,
        )
        logging.info(f"Updated dashboard {dashboard_id} to create version 2")

        # Step 4: Wait for version 2 to be ready
        v2_status = _wait_dashboard_version_status(
            quicksight_client, aws_account_id, dashboard_id,
            version_number=2,
            target_statuses=["CREATION_SUCCESSFUL"],
        )
        logging.info(f"Dashboard version 2 status: {v2_status}")

        # Verify the published version is still 1 (default describe returns published)
        resp = quicksight_client.describe_dashboard(
            AwsAccountId=aws_account_id,
            DashboardId=dashboard_id,
        )
        published_version = resp["Dashboard"]["Version"]["VersionNumber"]
        assert published_version == 1, (
            f"Expected published version 1, got {published_version}"
        )

        # Step 5: Adopt the dashboard into ACK
        resource_name = random_suffix_name("ack-adopt-dash-cr", 24)
        replacements = REPLACEMENT_VALUES.copy()
        replacements["DASHBOARD_NAME"] = resource_name
        replacements["DASHBOARD_ID"] = dashboard_id
        replacements["AWS_ACCOUNT_ID"] = aws_account_id

        resource_data = load_resource(
            "dashboard_adoption",
            additional_replacements=replacements,
        )

        ref = k8s.CustomResourceReference(
            CRD_GROUP, CRD_VERSION, DASHBOARD_RESOURCE_PLURAL,
            resource_name, namespace="default",
        )
        k8s.create_custom_resource(ref, resource_data)
        cr = k8s.wait_resource_consumed_by_controller(ref)
        assert cr is not None

        # Step 6: Wait for sync and verify version fields
        assert k8s.wait_on_condition(
            ref, "ACK.ResourceSynced", "True",
            wait_periods=DASHBOARD_SYNC_WAIT_PERIODS,
        )

        cr = k8s.get_resource(ref)
        assert "status" in cr

        cr_version_number = cr["status"].get("versionNumber")
        cr_version_status = cr["status"].get("versionStatus")

        logging.info(
            f"Adopted dashboard CR: versionNumber={cr_version_number}, "
            f"versionStatus={cr_version_status}"
        )

        # The adopted dashboard should reflect the published version (1),
        # not the latest draft version (2)
        assert cr_version_number == 1, (
            f"Expected adopted versionNumber to be 1 (published), "
            f"got {cr_version_number}"
        )
        assert cr_version_status == "CREATION_SUCCESSFUL", (
            f"Expected adopted versionStatus to be CREATION_SUCCESSFUL, "
            f"got {cr_version_status}"
        )

        # Step 7: Verify AWS still has version 1 as the published version
        # (controller did not publish version 2)
        resp = quicksight_client.describe_dashboard(
            AwsAccountId=aws_account_id,
            DashboardId=dashboard_id,
        )
        aws_published_version = resp["Dashboard"]["Version"]["VersionNumber"]
        assert aws_published_version == 1, (
            f"Expected AWS published version to remain 1 after adoption, "
            f"got {aws_published_version}"
        )

        # Cleanup: delete the K8s CR (deletion-policy: retain keeps the AWS resource)
        _, deleted = k8s.delete_custom_resource(ref, 3, 10)
        assert deleted

        # Clean up the AWS dashboard
        _delete_dashboard_via_boto(quicksight_client, dashboard_id, aws_account_id)
