from __future__ import annotations

import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[1]
CLI = ROOT / "json_validator.py"
SCHEMA = {
    "type": "object",
    "required": ["record_id", "status"],
    "properties": {"record_id": {"type": "string"}, "status": {"type": "string"}, "priority": {"type": "integer"}},
}


class JsonValidatorCliTests(unittest.TestCase):
    def invoke(self, document: object, schema: object = SCHEMA) -> subprocess.CompletedProcess[str]:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            input_path = root / "input.json"
            schema_path = root / "schema.json"
            input_path.write_text(json.dumps(document), encoding="utf-8")
            schema_path.write_text(json.dumps(schema), encoding="utf-8")
            return subprocess.run(
                [sys.executable, str(CLI), "--input", str(input_path), "--schema", str(schema_path)],
                check=False,
                capture_output=True,
                text=True,
            )

    def test_valid_document_has_canonical_stdout_and_zero_exit(self) -> None:
        result = self.invoke({"record_id": "alpha", "status": "OPEN", "priority": 1})
        self.assertEqual(result.returncode, 0)
        self.assertEqual(result.stderr, "")
        self.assertEqual(result.stdout, '{"errors":[],"valid":true}\n')

    def test_missing_required_field_is_invalid(self) -> None:
        result = self.invoke({"record_id": "alpha"})
        self.assertEqual(result.returncode, 1)
        self.assertEqual(json.loads(result.stdout), {"errors": ["missing_required_field:status"], "valid": False})

    def test_type_mismatch_is_invalid(self) -> None:
        result = self.invoke({"record_id": "alpha", "status": "OPEN", "priority": "high"})
        self.assertEqual(result.returncode, 1)
        self.assertEqual(json.loads(result.stdout), {"errors": ["type_mismatch:priority:integer"], "valid": False})

    def test_invalid_schema_returns_two(self) -> None:
        result = self.invoke({"record_id": "alpha"}, {"type": "array"})
        self.assertEqual(result.returncode, 2)
        self.assertEqual(json.loads(result.stdout), {"errors": ["schema_must_describe_an_object"], "valid": False})


if __name__ == "__main__":
    unittest.main()
