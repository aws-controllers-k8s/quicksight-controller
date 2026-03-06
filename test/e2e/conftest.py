# Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"). You may
# not use this file except in compliance with the License. A copy of the
# License is located at
#
#	 http://aws.amazon.com/apache2.0/
#
# or in the "license" file accompanying this file. This file is distributed
# on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
# express or implied. See the License for the specific language governing
# permissions and limitations under the License.

import boto3
import pytest
import logging

from acktest import k8s
from e2e.bootstrap_resources import get_bootstrap_resources

def pytest_addoption(parser):
    parser.addoption("--runslow", action="store_true", default=False, help="run slow tests")

def pytest_configure(config):
    config.addinivalue_line(
        "markers", "canary: mark test to also run in canary tests"
    )
    config.addinivalue_line(
        "markers", "service(arg): mark test associated with a given service"
    )
    config.addinivalue_line(
        "markers", "slow: mark test as slow to run"
    )

def pytest_collection_modifyitems(config, items):
    if config.getoption("--runslow"):
        return
    skip_slow = pytest.mark.skip(reason="need --runslow option to run")
    for item in items:
        if "slow" in item.keywords:
            item.add_marker(skip_slow)

@pytest.fixture(scope='session', autouse=True)
def quicksight_subscription():
    """Ensures QuickSight subscription is available from bootstrap resources.

    The subscription is created during the bootstrap phase (service_bootstrap.py).
    This fixture validates it was bootstrapped and exposes subscription info.
    """
    bootstrap_resources = get_bootstrap_resources()
    subscription = bootstrap_resources.QuickSightSubscription

    if subscription is None or not subscription.account_id:
        pytest.skip("QuickSight subscription not bootstrapped. Run service_bootstrap.py first.")

    logging.info(
        f"QuickSight subscription ready: account={subscription.account_id}, "
        f"edition={subscription.subscription_edition}"
    )

    yield {
        "AccountSubscriptionStatus": subscription.subscription_status,
        "Edition": subscription.subscription_edition,
    }

# Provide a k8s client to interact with the integration test cluster
@pytest.fixture(scope='class')
def k8s_client():
    return k8s._get_k8s_api_client()

@pytest.fixture(scope='module')
def quicksight_client(quicksight_subscription):
    """Returns a QuickSight client.

    Depends on quicksight_subscription to ensure subscription is ready before client is used.
    """
    return boto3.client('quicksight')

@pytest.fixture(scope='module')
def s3_client():
    return boto3.client('s3')
