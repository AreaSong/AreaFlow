#!/usr/bin/env python3
import json
import sys
from pathlib import Path


def type_matches(value, expected):
    if expected == "object":
        return isinstance(value, dict)
    if expected == "array":
        return isinstance(value, list)
    if expected == "string":
        return isinstance(value, str)
    if expected == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if expected == "boolean":
        return isinstance(value, bool)
    return True


def validate(schema, value, path="$"):
    expected_type = schema.get("type")
    if expected_type and not type_matches(value, expected_type):
        return [f"{path}: expected {expected_type}, got {type(value).__name__}"]

    errors = []
    if "const" in schema and value != schema["const"]:
        errors.append(f"{path}: expected const {schema['const']!r}, got {value!r}")

    if isinstance(value, str):
        min_length = schema.get("minLength")
        if min_length is not None and len(value) < min_length:
            errors.append(f"{path}: expected minLength {min_length}")

    if isinstance(value, int) and not isinstance(value, bool):
        minimum = schema.get("minimum")
        maximum = schema.get("maximum")
        if minimum is not None and value < minimum:
            errors.append(f"{path}: expected >= {minimum}, got {value}")
        if maximum is not None and value > maximum:
            errors.append(f"{path}: expected <= {maximum}, got {value}")

    if isinstance(value, dict):
        properties = schema.get("properties", {})
        for key in schema.get("required", []):
            if key not in value:
                errors.append(f"{path}: missing required property {key}")
        if schema.get("additionalProperties") is False:
            for key in value:
                if key not in properties:
                    errors.append(f"{path}: unexpected property {key}")
        for key, child_schema in properties.items():
            if key in value:
                errors.extend(validate(child_schema, value[key], f"{path}.{key}"))

    if isinstance(value, list):
        item_schema = schema.get("items")
        if item_schema:
            for index, item in enumerate(value):
                errors.extend(validate(item_schema, item, f"{path}[{index}]"))

    return errors


def main():
    if len(sys.argv) != 3:
        print("usage: validate-status-projection-schema.py <schema.json> <status.json>", file=sys.stderr)
        return 2

    schema_path = Path(sys.argv[1])
    status_path = Path(sys.argv[2])
    schema = json.loads(schema_path.read_text())
    status = json.loads(status_path.read_text())
    errors = validate(schema, status)
    if errors:
        print("status projection schema validation failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1
    print(f"status projection schema validation passed: {status_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
