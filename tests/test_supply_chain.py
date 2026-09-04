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
        for relative in supply_chain.INPUT_PATHS:
            path = root / relative
            path.parent.mkdir(parents=True, exist_ok=True)
            if relative.endswith(".yaml"):
                path.write_text("schema_version: 1.0.0\nrelease: v0.2.0\n", encoding="utf-8")
            elif relative.endswith(".json"):
                path.write_text("{}\n", encoding="utf-8")
            elif relative.endswith("go.mod"):
                path.write_text("module example.invalid/test\n\ngo 1.26\n", encoding="utf-8")
            else:
                path.write_text(f"dummy content for {relative}\n", encoding="utf-8")
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

    def test_seeded_tampered_provenance_fails_verification(self) -> None:
        root = self.make_root()
        fixture_path = pathlib.Path(__file__).resolve().parent / "fixtures" / "supply_chain" / "seeded_tampered_provenance.json"
        self.assertTrue(fixture_path.is_file(), "missing seeded provenance failure fixture")
        with tempfile.TemporaryDirectory() as output_name:
            output = pathlib.Path(output_name)
            supply_chain.write_bundle(root, output)
            (output / "provenance.json").write_bytes(fixture_path.read_bytes())
            self.assertFalse(supply_chain.verify_bundle(root, output))

    def test_seeded_tampered_sbom_fails_verification(self) -> None:
        root = self.make_root()
        fixture_path = pathlib.Path(__file__).resolve().parent / "fixtures" / "supply_chain" / "seeded_tampered_sbom.json"
        self.assertTrue(fixture_path.is_file(), "missing seeded sbom failure fixture")
        with tempfile.TemporaryDirectory() as output_name:
            output = pathlib.Path(output_name)
            supply_chain.write_bundle(root, output)
            (output / "sbom.spdx.json").write_bytes(fixture_path.read_bytes())
            self.assertFalse(supply_chain.verify_bundle(root, output))

    def test_seeded_tampered_platform_bom_fails_verification(self) -> None:
        root = self.make_root()
        fixture_path = pathlib.Path(__file__).resolve().parent / "fixtures" / "supply_chain" / "seeded_tampered_platform_bom.json"
        self.assertTrue(fixture_path.is_file(), "missing seeded platform-bom failure fixture")
        with tempfile.TemporaryDirectory() as output_name:
            output = pathlib.Path(output_name)
            supply_chain.write_bundle(root, output)
            (output / "platform-bom.json").write_bytes(fixture_path.read_bytes())
            self.assertFalse(supply_chain.verify_bundle(root, output))

    def test_missing_kernel_component_fails_closed(self) -> None:
        root = self.make_root()
        target = root / "modules" / "module-registry.yaml"
        self.assertTrue(target.is_file())
        target.unlink()
        with self.assertRaises(ValueError) as ctx:
            supply_chain.bundle(root)
        self.assertIn("modules/module-registry.yaml", str(ctx.exception))

    def test_kernel_components_cataloged_in_bom_and_sbom(self) -> None:
        root = self.make_root()
        artifacts = supply_chain.bundle(root)
        bom = __import__("json").loads(artifacts["platform-bom.json"])
        names = {item["name"] for item in bom["components"]}
        self.assertIn("modules/module-registry.yaml", names)
        self.assertIn("contracts/api/go.mod", names)
        self.assertIn("packages/identifiers/go.mod", names)
        self.assertIn("schemas/api/error-envelope.schema.json", names)
        self.assertIn("modules/organization-tenancy/go.mod", names)
        self.assertIn("modules/identity-authorization/go.mod", names)
        self.assertIn("modules/files-evidence/go.mod", names)
        self.assertIn("modules/records-audit/go.mod", names)
        self.assertIn("modules/configuration-checklist/go.mod", names)
        self.assertIn("modules/workflow-action/go.mod", names)
        self.assertIn("modules/events-outbox-jobs/go.mod", names)
        self.assertIn("modules/reporting-localization/go.mod", names)
        self.assertIn("modules/contract-migration-governance/go.mod", names)
        self.assertEqual(len(names), 18)


if __name__ == "__main__":
    unittest.main()
