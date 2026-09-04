#!/usr/bin/env python3
"""Deterministic, local-only JSON document validator for Candidate 2."""
from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


SUPPORTED_TYPES = {"array", "boolean", "integer", "null", "number", "object", "string"}


def emit(value: dict[str, Any]) -> None:
    print(json.dumps(value, ensure_ascii=True, separators=(",", ":"), sort_keys=True))


def read_json(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


def validate_schema(schema: Any) -> tuple[list[str], dict[str, str]]:
    if not isinstance(schema, dict) or schema.get("type") != "object":
        return ["schema_must_describe_an_object"], {}
    required = schema.get("required", [])
    properties = schema.get("properties", {})
    if not isinstance(required, list) or any(not isinstance(item, str) or not item for item in required):
        return ["schema_required_must_be_a_string_list"], {}
    if len(required) != len(set(required)):
        return ["schema_required_fields_must_be_unique"], {}
    if not isinstance(properties, dict):
        return ["schema_properties_must_be_an_object"], {}
    types: dict[str, str] = {}
    for field, definition in properties.items():
        if not isinstance(field, str) or not isinstance(definition, dict) or definition.get("type") not in SUPPORTED_TYPES:
            return ["schema_properties_must_use_supported_types"], {}
        types[field] = definition["type"]
    return [], types


def json_type(value: Any) -> str:
    if value is None:
        return "null"
    if isinstance(value, bool):
        return "boolean"
    if isinstance(value, int):
        return "integer"
    if isinstance(value, float):
        return "number"
    if isinstance(value, str):
        return "string"
    if isinstance(value, list):
        return "array"
    if isinstance(value, dict):
        return "object"
    raise TypeError("unsupported JSON value")


def validate_document(document: Any, schema: Any) -> tuple[int, dict[str, Any]]:
    schema_errors, property_types = validate_schema(schema)
    if schema_errors:
        return 2, {"errors": schema_errors, "valid": False}
    if not isinstance(document, dict):
        return 1, {"errors": ["document_must_be_an_object"], "valid": False}
    errors: list[str] = []
    for field in schema.get("required", []):
        if field not in document:
            errors.append(f"missing_required_field:{field}")
    for field, expected_type in property_types.items():
        if field in document and json_type(document[field]) != expected_type:
            errors.append(f"type_mismatch:{field}:{expected_type}")
    errors.sort()
    return (1 if errors else 0), {"errors": errors, "valid": not errors}


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate a JSON document against a deterministic local schema.")
    parser.add_argument("--input", required=True, type=Path)
    parser.add_argument("--schema", required=True, type=Path)
    args = parser.parse_args()
    try:
        document = read_json(args.input)
        schema = read_json(args.schema)
    except (OSError, json.JSONDecodeError) as error:
        emit({"errors": [f"input_or_schema_read_error:{type(error).__name__}"], "valid": False})
        return 2
    status, result = validate_document(document, schema)
    emit(result)
    return status


if __name__ == "__main__":
    raise SystemExit(main())
