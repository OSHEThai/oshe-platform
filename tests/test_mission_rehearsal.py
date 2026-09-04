from __future__ import annotations

import copy
import importlib.util
import pathlib
import sys
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).resolve().parents[1] / "tools" / "mission_rehearsal.py"
SPEC = importlib.util.spec_from_file_location("mission_rehearsal", MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
rehearsal = importlib.util.module_from_spec(SPEC)
sys.modules["mission_rehearsal"] = rehearsal
SPEC.loader.exec_module(rehearsal)
FIXTURE = pathlib.Path(__file__).resolve().parent / "fixtures" / "mission_rehearsal" / "offline_mission.json"


class MissionRehearsalTests(unittest.TestCase):
    def setUp(self) -> None:
        self.mission = rehearsal.load_mission(FIXTURE)

    def test_rehearsal_is_deterministic_and_uses_secretary_line(self) -> None:
        first, second = rehearsal.rehearse(self.mission), rehearsal.rehearse(copy.deepcopy(self.mission))
        self.assertEqual(first, second)
        self.assertEqual(first["status"], "COMPLETED")
        self.assertEqual(first["qualification"], "MOCK_ONLY_NOT_LIVE")
        self.assertTrue(any(event["owner_role"].endswith("Lead") and event["report_to"] == "PM Secretary" for event in first["trace"]))

    def test_direct_lead_to_pm_hidden_authority_and_unstructured_result_fail_closed(self) -> None:
        for field, value in (("report_to", "PM"), ("hidden_agent", True), ("result_contract", "FREE_TEXT")):
            with self.subTest(field=field):
                invalid = copy.deepcopy(self.mission)
                invalid["tasks"][1][field] = value
                with self.assertRaises(ValueError):
                    rehearsal.validate_mission(invalid)

    def test_concurrent_overlapping_writes_and_live_route_fail_closed(self) -> None:
        overlap = copy.deepcopy(self.mission)
        overlap["tasks"][2]["write_paths"] = ["work/implementation.json"]
        with self.assertRaisesRegex(ValueError, "overlap"):
            rehearsal.rehearse(overlap)
        route = copy.deepcopy(self.mission)
        route["tasks"][1]["route_id"] = "route-candidate"
        with self.assertRaisesRegex(ValueError, "live provider route"):
            rehearsal.validate_mission(route)

    def test_trace_collision_fails_closed(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            output = pathlib.Path(temporary) / "trace.json"
            rehearsal.write_trace(output, rehearsal.rehearse(self.mission))
            with self.assertRaises(FileExistsError):
                rehearsal.write_trace(output, rehearsal.rehearse(self.mission))


if __name__ == "__main__":
    unittest.main()
