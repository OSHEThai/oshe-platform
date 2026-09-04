from __future__ import annotations

import importlib.util
import pathlib
import subprocess
import sys
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).resolve().parents[1] / "tools" / "evidence_bundle.py"
SPEC = importlib.util.spec_from_file_location("evidence_bundle", MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
evidence_bundle = importlib.util.module_from_spec(SPEC)
sys.modules["evidence_bundle"] = evidence_bundle
SPEC.loader.exec_module(evidence_bundle)


class EvidenceBundleTests(unittest.TestCase):
    def make_root(self) -> tuple[pathlib.Path, str]:
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        root = pathlib.Path(temporary.name)
        files = {
            ".github/PULL_REQUEST_TEMPLATE.md": "## Mission and issue\n",
            ".ai/schemas/evidence-record.schema.json": "{}\n",
            ".ai/policies/evidence.yaml": "schema_version: 1\n",
            "repo-manifest.yaml": "repository:\n  name: oshe-platform\n",
            "README.md": "before\n",
        }
        for relative, content in files.items():
            path = root / relative
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content, encoding="utf-8")
        for args in (("init", "-q"), ("config", "user.email", "test@example.invalid"), ("config", "user.name", "test"), ("add", "."), ("commit", "-qm", "base")):
            subprocess.run(["git", *args], cwd=root, check=True)
        base = subprocess.run(["git", "rev-parse", "HEAD"], cwd=root, check=True, capture_output=True, text=True).stdout.strip()
        (root / "README.md").write_text("after\n", encoding="utf-8")
        subprocess.run(["git", "add", "README.md"], cwd=root, check=True)
        subprocess.run(["git", "commit", "-qm", "change"], cwd=root, check=True)
        return root, base

    def test_generation_is_deterministic_and_schema_shaped(self) -> None:
        root, base = self.make_root()
        with tempfile.TemporaryDirectory() as first_name, tempfile.TemporaryDirectory() as second_name:
            first, second = pathlib.Path(first_name), pathlib.Path(second_name)
            evidence_bundle.write_bundle(root, first, base, "V010-I028", "R2", evidence_bundle.DEFAULT_ALLOWED_PATHS)
            evidence_bundle.write_bundle(root, second, base, "V010-I028", "R2", evidence_bundle.DEFAULT_ALLOWED_PATHS)
            self.assertEqual({p.name: p.read_bytes() for p in first.iterdir()}, {p.name: p.read_bytes() for p in second.iterdir()})
            self.assertTrue(evidence_bundle.verify_bundle(root, first, base, "V010-I028", "R2", evidence_bundle.DEFAULT_ALLOWED_PATHS))
            record = __import__("json").loads((first / "evidence-record.yaml").read_bytes())
            self.assertEqual(record["schema_version"], "1.0.0")
            self.assertEqual(record["result"], "PASS")

    def test_out_of_bounds_change_fails_closed(self) -> None:
        root, base = self.make_root()
        with self.assertRaises(ValueError):
            evidence_bundle.bundle(root, base, "V010-I028", "R2", ("tools/",))

    def test_existing_bundle_is_never_replaced(self) -> None:
        root, base = self.make_root()
        with tempfile.TemporaryDirectory() as output_name:
            output = pathlib.Path(output_name)
            evidence_bundle.write_bundle(root, output, base, "V010-I028", "R2", evidence_bundle.DEFAULT_ALLOWED_PATHS)
            with self.assertRaises(FileExistsError):
                evidence_bundle.write_bundle(root, output, base, "V010-I028", "R2", evidence_bundle.DEFAULT_ALLOWED_PATHS)

    def test_missing_required_input_fails_closed(self) -> None:
        root, base = self.make_root()
        (root / ".ai" / "policies" / "evidence.yaml").unlink()
        with self.assertRaises(ValueError):
            evidence_bundle.bundle(root, base, "V010-I028", "R2", evidence_bundle.DEFAULT_ALLOWED_PATHS)


if __name__ == "__main__":
    unittest.main()
