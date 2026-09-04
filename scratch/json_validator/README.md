# Standalone JSON Validation CLI

Candidate 2 is a local-only deterministic validator. It reads one synthetic JSON document and one local schema, emits one canonical JSON result to standard output, and never contacts a provider or network service.

```powershell
python scratch/json_validator/json_validator.py --input input.json --schema schema.json
```

The supported schema is an object with `type: object`, optional string-array `required`, and optional `properties` entries using `array`, `boolean`, `integer`, `null`, `number`, `object`, or `string` types.

Exit code `0` means valid; `1` means a document validation failure; `2` means an unreadable input/schema or unsupported schema. Run the synthetic regression suite with:

```powershell
python -B -m unittest discover -s scratch/json_validator/tests -v
```
