from __future__ import annotations

import importlib.util
import pathlib
import sys
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).resolve().parents[1] / "tools" / "supply_chain.py"
SPEC = importlib.util.spec_from_file_location("supply_chain", MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
supply_chain = importlib.util.module_from_spec(SPEC)
sys.modules["supply_chain"] = supply_chain
SPEC.loader.exec_module(supply_chain)


class SupplyChainTests(unittest.TestCase):
    def make_root(self) -> pathlib.Path:
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        root = pathlib.Path(temporary.name)
        for relative, content in {
            ".ai/requirements-validation.txt": "example==1.0\n",
            "tools/dev/go.mod": "module example.invalid/tools\n\ngo 1.26\n",
            "toolchain.lock.yaml": "schema_version: 1\n",
            "repo-manifest.yaml": "repository:\n  release: v0.test\n",
            "LICENSE-POLICY.md": "No license.\n",
        }.items():
            path = root / relative
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content, encoding="utf-8")
        return root

    def test_generation_is_deterministic(self) -> None:
        root = self.make_root()
        with tempfile.TemporaryDirectory() as first_name, tempfile.TemporaryDirectory() as second_name:
            first, second = pathlib.Path(first_name), pathlib.Path(second_name)
            supply_chain.write_bundle(root, first)
            supply_chain.write_bundle(root, second)
            self.assertEqual(
                {path.name: path.read_bytes() for path in first.iterdir()},
                {path.name: path.read_bytes() for path in second.iterdir()},
            )
            self.assertTrue(supply_chain.verify_bundle(root, first))

    def test_tampered_artifact_fails_closed(self) -> None:
        root = self.make_root()
        with tempfile.TemporaryDirectory() as output_name:
            output = pathlib.Path(output_name)
            supply_chain.write_bundle(root, output)
            (output / "sbom.spdx.json").write_text("tampered\n", encoding="utf-8")
            self.assertFalse(supply_chain.verify_bundle(root, output))

    def test_existing_bundle_is_never_silently_replaced(self) -> None:
        root = self.make_root()
        with tempfile.TemporaryDirectory() as output_name:
            output = pathlib.Path(output_name)
            supply_chain.write_bundle(root, output)
            with self.assertRaises(FileExistsError):
                supply_chain.write_bundle(root, output)

    def test_signature_envelope_is_explicitly_unsigned_and_not_production_ready(self) -> None:
        root = self.make_root()
        envelope = __import__("json").loads(supply_chain.bundle(root)["signature-envelope.json"])
        self.assertEqual(envelope["kind"], "UNSIGNED_TEST_ONLY")
        self.assertEqual(envelope["algorithm"], "NONE")
        self.assertIsNone(envelope["signature"])
        self.assertFalse(envelope["production_ready"])


if __name__ == "__main__":
    unittest.main()
