from __future__ import annotations

import json
import pathlib
import unittest
from jsonschema import Draft202012Validator

ROOT = pathlib.Path(__file__).resolve().parents[1]
SCHEMA_PATH = ROOT / "schemas" / "api" / "error-envelope.schema.json"
FIXTURE_PATH = ROOT / "tests" / "fixtures" / "api_contracts" / "valid-error-response.json"


class ApiContractsStaticTests(unittest.TestCase):
    def setUp(self) -> None:
        self.assertTrue(SCHEMA_PATH.is_file(), f"missing schema at {SCHEMA_PATH}")
        self.assertTrue(FIXTURE_PATH.is_file(), f"missing fixture at {FIXTURE_PATH}")
        self.schema = json.loads(SCHEMA_PATH.read_text(encoding="utf-8"))
        self.fixture = json.loads(FIXTURE_PATH.read_text(encoding="utf-8"))
        Draft202012Validator.check_schema(self.schema)
        self.validator = Draft202012Validator(self.schema)

    def test_schema_validates_fixture(self) -> None:
        errors = list(self.validator.iter_errors(self.fixture))
        self.assertEqual(errors, [], f"fixture validation failed: {errors}")

    def test_required_fields_present_in_fixture(self) -> None:
        for required_field in ("code", "message", "correlation_id", "details"):
            self.assertIn(required_field, self.fixture)
            self.assertTrue(bool(self.fixture[required_field]))

    def test_missing_code_fails_validation(self) -> None:
        tampered = dict(self.fixture)
        del tampered["code"]
        self.assertFalse(self.validator.is_valid(tampered))

    def test_empty_correlation_id_fails_validation(self) -> None:
        tampered = dict(self.fixture)
        tampered["correlation_id"] = ""
        self.assertFalse(self.validator.is_valid(tampered))

    def test_additional_properties_prohibited(self) -> None:
        tampered = dict(self.fixture)
        tampered["unexpected_server_field"] = "malicious_payload"
        self.assertFalse(self.validator.is_valid(tampered))


if __name__ == "__main__":
    unittest.main()
