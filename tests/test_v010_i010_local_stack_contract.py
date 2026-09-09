from __future__ import annotations

import pathlib
import re
import subprocess
import sys
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))
from tools.verify_audit_provenance import ProvenanceTuple, parse_rfc_tuples, verify_tuple
COMPOSE = ROOT / "deploy" / "local" / "compose.dev.yaml"
ENV_EXAMPLE = ROOT / "deploy" / "local" / ".env.example"
SERVICES = ("postgres", "postgis", "meilisearch", "valkey", "seaweedfs", "nats")


class LocalStackContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.compose = COMPOSE.read_text(encoding="utf-8")
        cls.env = ENV_EXAMPLE.read_text(encoding="utf-8")

    def test_all_service_images_are_digest_pinned(self) -> None:
        images = re.findall(r"^\s*image:\s*(.+)$", self.compose, flags=re.MULTILINE)
        self.assertGreater(len(images), 0, "no image references found")
        for image in images:
            self.assertIn("@sha256:", image, f"tag-only/unpinned image: {image.strip()}")

    def test_every_service_declares_a_healthcheck(self) -> None:
        for name in SERVICES:
            self.assertRegex(
                self.compose,
                rf"(?m)^  {re.escape(name)}:(?:\n(?:    .*|      .*| {5,}.*))*?\n    healthcheck:",
                f"service missing healthcheck: {name}",
            )

    def test_missing_healthcheck_fails_negative_coverage(self) -> None:
        # Negative coverage test to ensure our healthcheck validation actually fails when it should
        bad_compose = "services:\n  postgres:\n    image: test\n  valkey:\n    image: test\n    healthcheck:\n      test: ping"
        with self.assertRaises(AssertionError):
            self.assertRegex(
                bad_compose,
                r"(?m)^  postgres:(?:\n(?:    .*|      .*| {5,}.*))*?\n    healthcheck:",
                "service missing healthcheck: postgres",
            )

    def test_no_production_like_settings(self) -> None:
        forbidden = re.compile(r"amazonaws\.com|\.cloud\.|production_endpoint|prod\.oshe|PROD_")
        self.assertNotRegex(self.compose, forbidden)
        self.assertNotRegex(self.env, forbidden)

    def test_authority_boundaries_encoded(self) -> None:
        self.assertIn("PostgreSQL is the AUTHORITATIVE transactional store", self.compose)
        self.assertIn("Meilisearch holds REBUILDABLE search projections only", self.compose)
        self.assertIn("Valkey is a REBUILDABLE, non-authoritative cache", self.compose)
        self.assertIn("NATS JetStream is messaging used only AFTER the transactional outbox", self.compose)
        self.assertIn("S3-compatible object store", self.compose)

    def test_no_direct_projection_authority_claim(self) -> None:
        for line in self.compose.splitlines():
            low = line.lower()
            if ("meilisearch" in low or "valkey" in low) and "authoritative" in low and "non-authoritative" not in low:
                self.fail(f"direct projection authority claim: {line.strip()}")

    def test_dockerfile_exact_digest_parity(self) -> None:
        dockerfile = (ROOT / ".devcontainer" / "Dockerfile").read_text(encoding="utf-8")
        
        # Parse actual Dockerfile FROM stage instructions exactly
        go_matches = re.findall(r"^FROM golang:1\.26\.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS go$", dockerfile, re.MULTILINE)
        self.assertEqual(len(go_matches), 1, "Expected exactly one valid go FROM instruction")

        node_matches = re.findall(r"^FROM node:24\.20\.0-alpine@sha256:e67514e5d0f6c46656005e1b693b2ec9d52e80b641307de684d4a015ba7a4eaf AS node$", dockerfile, re.MULTILINE)
        self.assertEqual(len(node_matches), 1, "Expected exactly one valid node FROM instruction")

        py_matches = re.findall(r"^FROM python:3\.14\.7-alpine@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc$", dockerfile, re.MULTILINE)
        self.assertEqual(len(py_matches), 1, "Expected exactly one valid python FROM instruction")

        all_froms = re.findall(r"^FROM .*$", dockerfile, re.MULTILINE)
        self.assertEqual(len(all_froms), 3, "Expected exactly three FROM instructions total")

    def test_dockerfile_digest_in_comment_fails_parity_negative_coverage(self) -> None:
        bad_dockerfile = "# FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS go\nFROM golang:1.26.5-alpine"
        matches = re.findall(r"^FROM golang:1\.26\.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS go$", bad_dockerfile, re.MULTILINE)
        self.assertEqual(len(matches), 0, "A digest in a comment should not satisfy parity")

    @staticmethod
    def _active_dockerfile_content(dockerfile: str) -> str:
        """Strips comment lines and empty lines to isolate active instructions."""
        return "\n".join(
            line for line in dockerfile.splitlines()
            if line.strip() and not line.strip().startswith("#")
        )

    @classmethod
    def _has_valid_corepack_launcher(cls, dockerfile: str) -> bool:
        active = cls._active_dockerfile_content(dockerfile)
        has_direct_copy = bool(
            re.search(r"^\s*COPY\s+--from=node\s+/usr/local/bin/corepack\s+/usr/local/bin/corepack\s*$", active, re.MULTILINE)
        )
        has_symlink_construction = bool(
            re.search(r"^\s*COPY\s+--from=node\s+/usr/local/lib/node_modules\s+/usr/local/lib/node_modules\s*$", active, re.MULTILINE)
            and re.search(r"^\s*COPY\s+--from=node\s+/usr/local/bin/node\s+/usr/local/bin/node\s*$", active, re.MULTILINE)
            and re.search(r"^\s*RUN\s+.*ln\s+-s\s+(?:\.\./lib/node_modules|/usr/local/lib/node_modules)/corepack/dist/corepack\.js\s+/usr/local/bin/corepack", active, re.MULTILINE)
        )
        return has_direct_copy or has_symlink_construction

    @classmethod
    def _has_valid_corepack_activation(cls, dockerfile: str) -> bool:
        active = cls._active_dockerfile_content(dockerfile)
        return bool(
            re.search(r"^\s*RUN\s+.*corepack\s+enable(?:\s+&&\s+corepack\s+prepare\s+pnpm@\d+\.\d+\.\d+\s+--activate)?", active, re.MULTILINE)
        )

    def test_devcontainer_viability_and_security(self) -> None:
        dockerfile = (ROOT / ".devcontainer" / "Dockerfile").read_text(encoding="utf-8")
        devcontainer = (ROOT / ".devcontainer" / "devcontainer.json").read_text(encoding="utf-8")
        
        import json
        dev_cfg = json.loads(devcontainer)
        self.assertEqual(dev_cfg.get("remoteUser"), "vscode", "remoteUser must be exactly vscode")
        
        self.assertRegex(dockerfile, r"(?s)addgroup -g 1000 vscode.*?adduser -u 1000 -G vscode", "vscode user not created")
        self.assertTrue(
            self._has_valid_corepack_launcher(dockerfile),
            "corepack launcher missing: expected supported active equivalent construction (direct COPY or symlink to corepack.js with node_modules)",
        )
        self.assertTrue(
            self._has_valid_corepack_activation(dockerfile),
            "corepack activation instruction missing or commented out",
        )
        
        self.assertNotIn("postCreateCommand", dev_cfg)
        self.assertNotIn("docker-cli", dockerfile)
        self.assertNotIn("/var/run/docker.sock", devcontainer)

    def test_devcontainer_remoteUser_negative_coverage(self) -> None:
        dev_cfg_bad1 = '{"name": "oshe-platform"}'
        import json
        with self.assertRaises(AssertionError):
            self.assertEqual(json.loads(dev_cfg_bad1).get("remoteUser"), "vscode")
            
        dev_cfg_bad2 = '{"name": "oshe-platform", "remoteUser": "root"}'
        with self.assertRaises(AssertionError):
            self.assertEqual(json.loads(dev_cfg_bad2).get("remoteUser"), "vscode")

    def test_devcontainer_corepack_launcher_negative_coverage(self) -> None:
        bad_dockerfile_missing = "FROM python:3.14.7-alpine\nRUN apk add git"
        with self.assertRaises(AssertionError):
            self.assertTrue(self._has_valid_corepack_launcher(bad_dockerfile_missing))

        # Incomplete construction: symlink present but node_modules copy missing
        bad_dockerfile_no_modules = (
            "COPY --from=node /usr/local/bin/node /usr/local/bin/node\n"
            "RUN ln -s ../lib/node_modules/corepack/dist/corepack.js /usr/local/bin/corepack\n"
        )
        with self.assertRaises(AssertionError):
            self.assertTrue(self._has_valid_corepack_launcher(bad_dockerfile_no_modules))

        # Incomplete construction: symlink target invalid
        bad_dockerfile_bad_target = (
            "COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules\n"
            "COPY --from=node /usr/local/bin/node /usr/local/bin/node\n"
            "RUN ln -s /some/invalid/path /usr/local/bin/corepack\n"
        )
        with self.assertRaises(AssertionError):
            self.assertTrue(self._has_valid_corepack_launcher(bad_dockerfile_bad_target))

        # Comment-only direct COPY must be rejected
        bad_dockerfile_comment_copy = (
            "# COPY --from=node /usr/local/bin/corepack /usr/local/bin/corepack\n"
            "FROM python:3.14.7-alpine\n"
        )
        with self.assertRaises(AssertionError):
            self.assertTrue(self._has_valid_corepack_launcher(bad_dockerfile_comment_copy))

        # Comment-only symlink line must be rejected
        bad_dockerfile_comment_symlink = (
            "COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules\n"
            "COPY --from=node /usr/local/bin/node /usr/local/bin/node\n"
            "# RUN ln -s ../lib/node_modules/corepack/dist/corepack.js /usr/local/bin/corepack\n"
        )
        with self.assertRaises(AssertionError):
            self.assertTrue(self._has_valid_corepack_launcher(bad_dockerfile_comment_symlink))

        # Comment-only activation line must be rejected
        bad_dockerfile_comment_activation = (
            "COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules\n"
            "COPY --from=node /usr/local/bin/node /usr/local/bin/node\n"
            "RUN ln -s ../lib/node_modules/corepack/dist/corepack.js /usr/local/bin/corepack\n"
            "# RUN apk add --no-cache libstdc++ libgcc && corepack enable && corepack prepare pnpm@11.24.0 --activate\n"
        )
        with self.assertRaises(AssertionError):
            self.assertTrue(self._has_valid_corepack_activation(bad_dockerfile_comment_activation))

    def test_fail_closed_native_compose_regression(self) -> None:
        bootstrap = (ROOT / "deploy" / "local" / "bootstrap.ps1").read_text(encoding="utf-8")
        teardown = (ROOT / "deploy" / "local" / "teardown-rebuild.ps1").read_text(encoding="utf-8")
        
        # Check fail-closed ordering
        self.assertRegex(bootstrap, r"(?m)^docker compose.*$\n^if \(\$LASTEXITCODE -ne 0\) \{ throw .* \}$")
        self.assertRegex(teardown, r"(?m)^docker compose -f .*? down -v$\n^if \(\$LASTEXITCODE -ne 0\) \{ throw .* \}$")
        self.assertRegex(teardown, r"(?m)^docker compose -f .*? up -d --wait$\n^if \(\$LASTEXITCODE -ne 0\) \{ throw .* \}$")
        
        # Check order before seed and completion
        self.assertRegex(bootstrap, r"(?s)if \(\$LASTEXITCODE -ne 0\).*?seed-synthetic.*COMPLETE")
        self.assertRegex(teardown, r"(?s)if \(\$LASTEXITCODE -ne 0\).*?docker compose.*up.*?if \(\$LASTEXITCODE -ne 0\).*?COMPLETE")

    def test_fail_closed_negative_mutations(self) -> None:
        bad_bootstrap_write = "docker compose up\nif ($LASTEXITCODE -ne 0) { Write-Output 'Failed' }"
        with self.assertRaises(AssertionError):
            self.assertRegex(bad_bootstrap_write, r"(?m)^docker compose.*$\n^if \(\$LASTEXITCODE -ne 0\) \{ throw .* \}$")
            
        bad_bootstrap_eq = "docker compose up\nif ($LASTEXITCODE -eq 0) { throw 'Failed' }"
        with self.assertRaises(AssertionError):
            self.assertRegex(bad_bootstrap_eq, r"(?m)^docker compose.*$\n^if \(\$LASTEXITCODE -ne 0\) \{ throw .* \}$")

    def test_rfc_provenance_tuples_structure_and_parseability(self) -> None:
        rfc_file = ROOT / "docs" / "rfc" / "audit-20260908-baseline-contract-reconciliation.md"
        self.assertTrue(rfc_file.is_file(), f"missing RFC file: {rfc_file}")
        tuples = parse_rfc_tuples(rfc_file.read_text(encoding="utf-8"))
        self.assertGreaterEqual(len(tuples), 2, "expected at least compose and seed tuples in RFC")
        targets = {t.target for t in tuples}
        self.assertIn("compose", targets)
        self.assertIn("seed", targets)
        for t in tuples:
            self.assertRegex(t.commit, r"^[0-9a-f]{40}$", f"invalid commit SHA in tuple: {t}")
            self.assertRegex(t.blob, r"^[0-9a-f]{40}$", f"invalid blob SHA in tuple: {t}")
            self.assertRegex(t.normalized_sha256, r"^[0-9a-f]{64}$", f"invalid normalized SHA-256 in tuple: {t}")
            self.assertTrue(t.path.startswith("deploy/local/"), f"unexpected path in tuple: {t}")

    def test_hermetic_provenance_valid_mock(self) -> None:
        sample = ProvenanceTuple(
            target="compose",
            commit="d36ff7d6495fff954ce645f6a0d7743b85b77c17",
            path="deploy/local/compose.dev.yaml",
            blob="958100545e76652235123e5a85eb8696006706de",
            normalized_sha256="53ab5ff03bd4fa90fec648b62b6a8126aa581ca94bb1a6300cc423fed7174a13",
        )
        raw_bytes = (ROOT / "deploy" / "local" / "compose.dev.yaml").read_bytes()
        def mock_valid(cmd: list[str]) -> subprocess.CompletedProcess:
            if "^{commit}" in cmd[3]:
                return subprocess.CompletedProcess(cmd, 0, stdout=b"d36ff7d6495fff954ce645f6a0d7743b85b77c17\n", stderr=b"")
            if f"{sample.commit}:{sample.path}" in cmd[3]:
                return subprocess.CompletedProcess(cmd, 0, stdout=b"958100545e76652235123e5a85eb8696006706de\n", stderr=b"")
            if cmd[1:3] == ["cat-file", "-t"]:
                return subprocess.CompletedProcess(cmd, 0, stdout=b"blob\n", stderr=b"")
            if cmd[1:3] == ["cat-file", "-p"]:
                return subprocess.CompletedProcess(cmd, 0, stdout=raw_bytes, stderr=b"")
            return subprocess.CompletedProcess(cmd, 1, stdout=b"", stderr=b"unknown cmd")

        errors = verify_tuple(sample, repo_root=ROOT, git_runner=mock_valid)
        self.assertEqual(errors, [], f"expected zero errors for valid tuple, got: {errors}")

    def test_hermetic_provenance_missing_commit_mock(self) -> None:
        sample = ProvenanceTuple(
            target="compose",
            commit="d36ff7d6495fff954ce645f6a0d7743b85b77c17",
            path="deploy/local/compose.dev.yaml",
            blob="958100545e76652235123e5a85eb8696006706de",
            normalized_sha256="53ab5ff03bd4fa90fec648b62b6a8126aa581ca94bb1a6300cc423fed7174a13",
        )
        def mock_missing_commit(cmd: list[str]) -> subprocess.CompletedProcess:
            return subprocess.CompletedProcess(cmd, 1, stdout=b"", stderr=b"fatal: Not a valid object name")

        errors = verify_tuple(sample, repo_root=ROOT, git_runner=mock_missing_commit)
        self.assertEqual(len(errors), 1)
        self.assertIn("missing from local git history", errors[0])

    def test_hermetic_provenance_missing_path_mock(self) -> None:
        sample = ProvenanceTuple(
            target="compose",
            commit="d36ff7d6495fff954ce645f6a0d7743b85b77c17",
            path="deploy/local/nonexistent.yaml",
            blob="958100545e76652235123e5a85eb8696006706de",
            normalized_sha256="53ab5ff03bd4fa90fec648b62b6a8126aa581ca94bb1a6300cc423fed7174a13",
        )
        def mock_missing_path(cmd: list[str]) -> subprocess.CompletedProcess:
            if "^{commit}" in cmd[3]:
                return subprocess.CompletedProcess(cmd, 0, stdout=b"d36ff7d6495fff954ce645f6a0d7743b85b77c17\n", stderr=b"")
            return subprocess.CompletedProcess(cmd, 1, stdout=b"", stderr=b"fatal: path not found")

        errors = verify_tuple(sample, repo_root=ROOT, git_runner=mock_missing_path)
        self.assertEqual(len(errors), 1)
        self.assertIn("failed to resolve path", errors[0])

    def test_hermetic_provenance_mismatched_blob_mock(self) -> None:
        sample = ProvenanceTuple(
            target="compose",
            commit="d36ff7d6495fff954ce645f6a0d7743b85b77c17",
            path="deploy/local/compose.dev.yaml",
            blob="958100545e76652235123e5a85eb8696006706de",
            normalized_sha256="53ab5ff03bd4fa90fec648b62b6a8126aa581ca94bb1a6300cc423fed7174a13",
        )
        def mock_mismatched_blob(cmd: list[str]) -> subprocess.CompletedProcess:
            if "^{commit}" in cmd[3]:
                return subprocess.CompletedProcess(cmd, 0, stdout=b"d36ff7d6495fff954ce645f6a0d7743b85b77c17\n", stderr=b"")
            if f"{sample.commit}:{sample.path}" in cmd[3]:
                return subprocess.CompletedProcess(cmd, 0, stdout=b"0000000000000000000000000000000000000000\n", stderr=b"")
            return subprocess.CompletedProcess(cmd, 0, stdout=b"", stderr=b"")

        errors = verify_tuple(sample, repo_root=ROOT, git_runner=mock_mismatched_blob)
        self.assertEqual(len(errors), 1)
        self.assertIn("blob mismatch", errors[0])

    def test_hermetic_provenance_mismatched_hash_mock(self) -> None:
        sample = ProvenanceTuple(
            target="compose",
            commit="d36ff7d6495fff954ce645f6a0d7743b85b77c17",
            path="deploy/local/compose.dev.yaml",
            blob="958100545e76652235123e5a85eb8696006706de",
            normalized_sha256="53ab5ff03bd4fa90fec648b62b6a8126aa581ca94bb1a6300cc423fed7174a13",
        )
        def mock_mismatched_hash(cmd: list[str]) -> subprocess.CompletedProcess:
            if "^{commit}" in cmd[3]:
                return subprocess.CompletedProcess(cmd, 0, stdout=b"d36ff7d6495fff954ce645f6a0d7743b85b77c17\n", stderr=b"")
            if f"{sample.commit}:{sample.path}" in cmd[3]:
                return subprocess.CompletedProcess(cmd, 0, stdout=b"958100545e76652235123e5a85eb8696006706de\n", stderr=b"")
            if cmd[1:3] == ["cat-file", "-t"]:
                return subprocess.CompletedProcess(cmd, 0, stdout=b"blob\n", stderr=b"")
            if cmd[1:3] == ["cat-file", "-p"]:
                return subprocess.CompletedProcess(cmd, 0, stdout=b"tampered content without valid hash\n", stderr=b"")
            return subprocess.CompletedProcess(cmd, 0, stdout=b"", stderr=b"")

        errors = verify_tuple(sample, repo_root=ROOT, git_runner=mock_mismatched_hash)
        self.assertEqual(len(errors), 1)
        self.assertIn("normalized SHA-256 mismatch", errors[0])

    def test_hermetic_provenance_unparseable_rfc_content(self) -> None:
        empty_rfc = "# Empty RFC\nNo tuples here.\n"
        with self.assertRaises(ValueError):
            parse_rfc_tuples(empty_rfc)


if __name__ == "__main__":
    unittest.main()
