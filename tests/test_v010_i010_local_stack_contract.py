from __future__ import annotations

import pathlib
import re
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
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

    def test_devcontainer_viability_and_security(self) -> None:
        dockerfile = (ROOT / ".devcontainer" / "Dockerfile").read_text(encoding="utf-8")
        devcontainer = (ROOT / ".devcontainer" / "devcontainer.json").read_text(encoding="utf-8")
        
        import json
        dev_cfg = json.loads(devcontainer)
        self.assertEqual(dev_cfg.get("remoteUser"), "vscode", "remoteUser must be exactly vscode")
        
        self.assertRegex(dockerfile, r"(?s)addgroup -g 1000 vscode.*?adduser -u 1000 -G vscode", "vscode user not created")
        self.assertIn("COPY --from=node /usr/local/bin/corepack /usr/local/bin/corepack", dockerfile, "corepack launcher missing")
        
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


if __name__ == "__main__":
    unittest.main()
