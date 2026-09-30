import unittest

from platform_readiness import monitoring_targets_ready


class MonitoringReadinessTests(unittest.TestCase):
    def test_discovery_must_include_expected_storage_target(self):
        existing = [{"labels": {"service": "existing"}, "health": "up"}] * 26
        storage = {"labels": {"service": "seaweedfs", "namespace": "atlas-storage"}, "health": "up"}
        self.assertFalse(monitoring_targets_ready(existing))
        self.assertTrue(monitoring_targets_ready(existing + [storage]))
        self.assertFalse(monitoring_targets_ready(existing + [storage, storage]))

    def test_health_and_identity_remain_mandatory(self):
        storage = {"labels": {"service": "seaweedfs", "namespace": "atlas-storage"}, "health": "up"}
        self.assertFalse(monitoring_targets_ready([]))
        self.assertFalse(monitoring_targets_ready([dict(storage, health="down")]))
        self.assertFalse(monitoring_targets_ready([dict(storage, health=None)]))
        self.assertFalse(monitoring_targets_ready([dict(storage, labels={"service": "seaweedfs", "namespace": "other"})]))
        self.assertFalse(monitoring_targets_ready([storage, {"health": "unknown"}]))
