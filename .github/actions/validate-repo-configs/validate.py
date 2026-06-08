import json
import sys
import glob
import os
import yaml
import jsonschema


def resolve_schema(config_path, fallback_schema_path):
    config_root = os.path.realpath(config_path)
    override = os.path.realpath(os.path.join(config_path, ".schemas", "repository-config.schema.json"))

    if not override.startswith(config_root):
        print(f"Error: schema override path escapes config directory: {override}")
        sys.exit(1)

    if os.path.exists(override):
        print(f"Using org schema override: {override}")
        return override

    print(f"Using default schema: {fallback_schema_path}")
    return fallback_schema_path


def validate(config_path, fallback_schema_path):
    schema_path = resolve_schema(config_path, fallback_schema_path)

    try:
        with open(schema_path) as f:
            schema = json.load(f)
    except (OSError, json.JSONDecodeError) as e:
        print(f"Error: failed to load schema from {schema_path}: {e}")
        sys.exit(1)

    files = glob.glob(os.path.join(config_path, "repos", "*.yaml"))
    if not files:
        print("No repos/*.yaml files found, skipping validation")
        return []

    errors = []
    for f in files:
        with open(f) as fh:
            data = yaml.safe_load(fh)
        try:
            jsonschema.validate(data, schema)
            print(f"ok -- {f}")
        except jsonschema.ValidationError as e:
            errors.append(f"{f}: {e.message} at {e.json_path}")

    return errors


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: validate.py <config-path> <fallback-schema-path>")
        sys.exit(1)

    validation_errors = validate(sys.argv[1], sys.argv[2])

    if validation_errors:
        print("\nSchema validation errors were encountered:")
        for err in validation_errors:
            print(f"  {err}")
        sys.exit(1)
