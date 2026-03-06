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
"""Bootstraps the resources required to run the QuickSight integration tests."""

import logging
import os

from acktest.bootstrapping import Resources, BootstrapFailureException
from acktest.bootstrapping.iam import Role
from acktest.bootstrapping.quicksight import Subscription, S3DataSource

from e2e import bootstrap_directory
from e2e.bootstrap_resources import BootstrapResources


def service_bootstrap() -> Resources:
    logging.getLogger().setLevel(logging.INFO)

    notification_email = os.environ.get(
        "QUICKSIGHT_NOTIFICATION_EMAIL", "ack-infra+quicksight-resources@amazon.com"
    )
    edition = os.environ.get("QUICKSIGHT_EDITION", "ENTERPRISE")

    resources = BootstrapResources(
        QuickSightSubscription=Subscription(
            notification_email=notification_email,
            edition=edition,
        ),
        DataSource=S3DataSource(name_prefix="qs-test-bucket"),
        QuickSightS3Role=Role(
            name_prefix="qs-test-role",
            principal_service="quicksight.amazonaws.com",
            description="Test role for QuickSight DataSource e2e tests",
            managed_policies=["arn:aws:iam::aws:policy/AdministratorAccess"],
        ),
    )

    try:
        resources.bootstrap()
    except BootstrapFailureException as ex:
        exit(254)

    return resources


if __name__ == "__main__":
    config = service_bootstrap()
    config.serialize(bootstrap_directory)
