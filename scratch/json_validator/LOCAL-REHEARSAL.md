# Local Mission Rehearsal

`mission-dag.json` records a deterministic local-only preflight, injected failure, and recovery rehearsal for Candidate 2.

Run it with:

```powershell
python -B -m unittest discover -s scratch/json_validator/tests -v
```

The rehearsal proves only the deterministic JSON validator behavior against synthetic files. It is not evidence of a complete Herdr mission, cross-route comparison, provider activation or qualification, release, or deployment.
