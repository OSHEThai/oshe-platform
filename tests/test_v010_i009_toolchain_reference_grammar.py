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


if __name__ == "__main__":
    unittest.main()
