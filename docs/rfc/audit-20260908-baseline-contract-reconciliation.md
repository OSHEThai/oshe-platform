# RFC: Audit 2026-09-08 Baseline Contract Reconciliation

## 1. Objective
Reconcile baseline contract drift exposed during the CI coverage audit (Lane C), strictly isolating the changes to the pinned baseline c606876f209daa4c361a086e817ba20733c79b44 without weakening any fail-closed governance behaviors.

## 2. Classifier Git Provenance & Tamper Checks
**Finding:** Classifier sealed hashes drifted from canonical Git provenance (old seal from 45b5e44).
**Resolution:** 
- Verified exact Git provenance of the canonical compose file: commit d36ff7d6495fff954ce645f6a0d7743b85b77c17 (PR1008), blob 958100545e76652235123e5a85eb8696006706de.
- New normalized SHA-256 for compose is 53ab5ff03bd4fa90fec648b62b6a8126aa581ca94bb1a6300cc423fed7174a13.
- Verified exact Git provenance of the canonical seed file: commit 951bb91caa2621bbb46d10f3030f7df3eecaf71f, blob c788fe0ff090aa65217df5894a0ae48df5a22db4 (unchanged).
- Restored exact digest verification matching these authoritative historical baseline proofs. 
- Maintained complete fail-closed behavior for tampering; digests were updated based on explicit reviewed provenance, not merely matching current bytes.

## 3. Database Ordering Module Snapshots
**Finding:** Database v020 snapshot expected nine modules but drifted due to later registry additions.
**Resolution:**
- Enforced that the v020 snapshot strictly remains the exact nine-module manifest.
- Distinguished later registry additions systematically, explicitly blocking them from receiving migration runtime approval within the v020 lifecycle.
- Strengthened negative DAG uniqueness and validation checks to prevent future leakage of unauthorized modules into the locked snapshot.

## 4. Corepack Launcher Construction
**Finding:** Tests asserted a stale literal COPY of the corepack launcher rather than valid behavior.
**Resolution:**
- Replaced the literal COPY assertion with a test of the actual pinned devcontainer corepack launcher's supported equivalent construction.
- Executed completely isolated from Docker or network state, ensuring the test remains an exact offline structural verification.

## 5. Security Validation Request
This RFC and associated test changes are submitted for Security review by p13 to confirm no fail-closed validation has been bypassed.
