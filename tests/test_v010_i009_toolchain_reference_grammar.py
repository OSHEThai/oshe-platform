from __future__ import annotations

import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


def load_embedded_checker(script_path: pathlib.Path, start: str, end: str) -> dict[str, object]:
    text = script_path.read_text(encoding="utf-8")
    python_code = text.split(start, 1)[1].split(end, 1)[0]
    definitions = python_code.split("\n\ntry:\n    main()", 1)[0]
    namespace: dict[str, object] = {"__name__": "embedded_checker_test"}
    exec(definitions, namespace)
    return namespace


class StrictReferenceGrammarTests(unittest.TestCase):
    def setUp(self) -> None:
        self.checkers = {
            "shell": load_embedded_checker(
                ROOT / ".ci/checks/v010-i009-toolchain.sh", "<<'PY'\n", "\nPY\n"
            ),
            "powershell": load_embedded_checker(
                ROOT / ".ci/checks/v010-i009-toolchain.ps1", "$pythonCode = @'\n", "\n'@\n"
            ),
        }

    def assert_rejected(self, value: str, path: str) -> None:
        for name, checker in self.checkers.items():
            with self.subTest(checker=name, value=value):
                with self.assertRaises(checker["ContractError"]):
                    checker["inspect_scalars"](value, path)

    def test_rejects_unpinned_references_ranges_and_documented_aliases(self) -> None:
        for value in ("nginx", "nginx:1.25"):
            self.assert_rejected(value, "local_services.container_image")
        for value in ("*", "1.*", "1.2.*", "1.2.x"):
            self.assert_rejected(value, "frontend_dependencies.react")
        for value in ("latest", "stable", "edge", "rolling", "canary", "main", "master", "dev", "nightly"):
            self.assert_rejected(value, "local_services.container_image")

    def test_field_aware_allowlist_preserves_selected_fixed_value(self) -> None:
        for name, checker in self.checkers.items():
            with self.subTest(checker=name):
                checker["inspect_scalars"]("4.29", "local_services.seaweedfs")

    def test_postgis_image_identity_is_digest_pinned(self) -> None:
        digest = "sha256:a8ffa9afeea4ad6eada171fa2afdb57cd3eb90f92ce20156aa2cb8411d70e0cd"
        pinned = f"postgis/postgis:17-3.6-alpine@{digest}"
        for name, checker in self.checkers.items():
            with self.subTest(checker=name):
                checker["inspect_scalars"](pinned, "local_services.postgis.image_ref")
                checker["inspect_scalars"]("linux/amd64", "local_services.postgis.platform")
                with self.assertRaises(checker["ContractError"]):
                    checker["inspect_scalars"](
                        "postgis/postgis:17-3.6-alpine", "local_services.postgis.image_ref"
                    )

    def test_all_local_service_images_are_digest_pinned(self) -> None:
        services = {
            "postgresql": ("postgres:17.11-alpine", "sha256:18cfe3ef5e6815560c98237d6216d1e5119702fb0f3894c8785dd58b8bbe5d73"),
            "meilisearch": ("getmeili/meilisearch:v1.51.0", "sha256:a9eb29ee09ab4943db3b4c68620bd6f3382e6b2b0ac4431c0e607b48dbcd4c14"),
            "valkey": ("valkey/valkey:9.1.1-alpine", "sha256:15568b9cb7eb67f4aed4de018c23f13d344e0e6437b31fe8fb8823dc81ebb3a9"),
            "seaweedfs": ("chrislusf/seaweedfs:4.29", "sha256:d47c7ee99fcb951351d7194915f4e3a5ea604a8e8871183d713907dec4fb9bf5"),
            "nats_jetstream": ("nats:2.14.5-alpine", "sha256:d4ac35882ac65aff236cd65b9d3fa4d24332c681e1a85f94eedccd3cdd65b1da"),
        }
        for service, (tag_ref, digest) in services.items():
            pinned = f"{tag_ref}@{digest}"
            for name, checker in self.checkers.items():
                with self.subTest(checker=name, service=service):
                    checker["inspect_scalars"](pinned, f"local_services.{service}.image_ref")
                    checker["inspect_scalars"]("linux/amd64", f"local_services.{service}.platform")
                    with self.assertRaises(checker["ContractError"]):
                        checker["inspect_scalars"](tag_ref, f"local_services.{service}.image_ref")


if __name__ == "__main__":
    unittest.main()
