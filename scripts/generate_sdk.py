#!/usr/bin/env python3
"""Generate the typed Fanvue SDK from a vendored OpenAPI 3.1 specification.

The Fanvue API publishes a single consolidated OpenAPI document at
``https://api.fanvue.com/docs/openapi.json``. A snapshot of that document is
vendored into ``fanvue_sdk/_spec/openapi.json`` so generation is fully
reproducible and works offline. By default the generator reads the vendored
snapshot; pass ``--refresh`` to re-download the latest spec and overwrite the
snapshot before generating.

Outputs (all marked auto-generated):

* ``fanvue_sdk/_operations.py`` — operation metadata registry.
* ``fanvue_sdk/models.py`` — Pydantic response models + per-operation adapters.
* ``fanvue_sdk/resources/*.py`` — typed resource classes (except ``base.py``).
* ``fanvue_sdk/resources/__init__.py`` — resource registry.
* ``docs-endpoints.md`` — human-readable operation index.

This module never edits hand-maintained code (the client, exceptions, types,
or ``resources/base.py``).
"""

from __future__ import annotations

import argparse
import json
import keyword
import re
import sys
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Any
from urllib.request import urlopen

ROOT = Path(__file__).resolve().parents[1]
PACKAGE_DIR = ROOT / "fanvue_sdk"
RESOURCES_DIR = PACKAGE_DIR / "resources"
SPEC_DIR = PACKAGE_DIR / "_spec"
SPEC_PATH = SPEC_DIR / "openapi.json"

OPENAPI_URL = "https://api.fanvue.com/docs/openapi.json"
DOCS_REFERENCE_BASE = "https://api.fanvue.com/docs/api-reference/reference"

PATH_PARAMETER_PATTERN = re.compile(r"\{([^}]+)\}")
HTTP_METHODS = {"get", "post", "put", "patch", "delete", "options", "head"}

# Header parameters the SDK injects itself; never surfaced on method signatures.
SDK_MANAGED_HEADERS = {"authorization", "x-fanvue-api-version"}


@dataclass(slots=True)
class Param:
    """A single OpenAPI parameter rendered for the SDK."""

    name: str
    python_name: str
    location: str
    required: bool
    annotation: str


@dataclass(slots=True)
class Operation:
    """A fully-resolved SDK operation."""

    operation_id: str
    group: str
    method: str
    path: str
    summary: str
    description: str
    docs_url: str
    parameters: list[Param]
    request_body_required: bool
    request_body_content_types: tuple[str, ...]
    response_type_name: str


@dataclass(slots=True)
class ModelField:
    """A single field on a generated Pydantic model."""

    original_name: str
    python_name: str
    annotation: str
    required: bool
    alias: str | None


@dataclass(slots=True)
class ModelDefinition:
    """A generated Pydantic model and its ordered fields."""

    name: str
    fields: list[ModelField]


@dataclass(slots=True)
class ModelBuildState:
    """Mutable accumulator threaded through schema resolution."""

    used_type_names: set[str]
    model_definitions: dict[str, ModelDefinition]
    model_order: list[str]
    response_type_aliases: dict[str, str]
    response_alias_order: list[str]
    operation_response_aliases: dict[str, str]
    requires_literal: bool
    uses_field_alias: bool

    def reserve_type_name(self, base_name: str) -> str:
        normalized = pascal_case(base_name) or "GeneratedType"
        candidate = normalized
        index = 2
        while candidate in self.used_type_names:
            candidate = f"{normalized}{index}"
            index += 1
        self.used_type_names.add(candidate)
        return candidate


class SchemaResolver:
    """Resolves OpenAPI schemas into Python type annotations + Pydantic models.

    A single resolver instance is shared across the whole spec so that
    ``#/components/schemas`` references resolve to one model definition reused
    by every operation that references it.
    """

    def __init__(self, *, state: ModelBuildState, components: dict[str, Any]) -> None:
        self._state = state
        self._components = components
        self._resolved_component_types: dict[str, str] = {}
        self._component_model_names: dict[str, str] = {}
        self._resolving_refs: set[str] = set()

    def resolve_schema(self, schema: dict[str, Any] | None, hint: str) -> str:
        if not isinstance(schema, dict):
            return "Any"

        ref = schema.get("$ref")
        if isinstance(ref, str):
            return self._resolve_ref(ref)

        # Honour the legacy ``nullable: true`` keyword (the Fanvue spec is
        # declared OpenAPI 3.1 but uses the Swagger 2.0 / OpenAPI 3.0 form).
        nullable = schema.get("nullable") is True

        for combinator in ("oneOf", "anyOf", "allOf"):
            variants = schema.get(combinator)
            if isinstance(variants, list):
                return add_optional(self._resolve_union(variants, hint), nullable)

        schema_type: Any = schema.get("type")
        if isinstance(schema_type, list):
            nullable = nullable or "null" in schema_type
            non_null = [entry for entry in schema_type if entry != "null"]
            schema_type = non_null[0] if len(non_null) == 1 else None
        elif schema_type == "null":
            return "None"

        enum_values = extract_enum_values(schema.get("enum"))
        if enum_values:
            filtered = [value for value in enum_values if value is not None]
            if len(filtered) != len(enum_values):
                nullable = True
            if filtered:
                self._state.requires_literal = True
                literal = ", ".join(repr(value) for value in deduplicate(filtered))
                annotation = f"Literal[{literal}]"
            else:
                annotation = "Any"
            return add_optional(annotation, nullable)

        if schema_type == "array":
            item_schema = schema.get("items") if isinstance(schema.get("items"), dict) else None
            item_type = self.resolve_schema(item_schema, f"{hint}Item")
            return add_optional(f"list[{item_type}]", nullable)

        if self._is_mapping_schema(schema):
            additional = schema.get("additionalProperties")
            if isinstance(additional, dict):
                value_type = self.resolve_schema(additional, f"{hint}Value")
            else:
                value_type = "Any"
            return add_optional(f"dict[str, {value_type}]", nullable)

        if self._is_model_schema(schema):
            model_name = self._state.reserve_type_name(hint)
            self._build_model(model_name, schema)
            return add_optional(model_name, nullable)

        return add_optional(primitive_schema_type(schema_type), nullable)

    def _resolve_union(self, variants: list[Any], hint: str) -> str:
        options: list[str] = []
        for index, option in enumerate(variants, start=1):
            option_schema = option if isinstance(option, dict) else None
            options.append(self.resolve_schema(option_schema, f"{hint}Option{index}"))
        compact = deduplicate(options)
        if not compact:
            return "Any"
        return " | ".join(compact)

    def _resolve_ref(self, ref: str) -> str:
        prefix = "#/components/schemas/"
        if not ref.startswith(prefix):
            return "Any"

        schema_name = ref[len(prefix) :]
        if schema_name in self._resolved_component_types:
            return self._resolved_component_types[schema_name]

        schema = self._components.get(schema_name)
        if not isinstance(schema, dict):
            return "Any"

        if schema_name in self._resolving_refs:
            return self._component_model_names.get(schema_name, "Any")

        self._resolving_refs.add(schema_name)
        try:
            if self._is_model_schema(schema):
                model_name = self._component_model_names.get(schema_name)
                if model_name is None:
                    model_name = self._state.reserve_type_name(pascal_case(schema_name))
                    self._component_model_names[schema_name] = model_name
                self._resolved_component_types[schema_name] = model_name
                self._build_model(model_name, schema)
                return model_name

            resolved = self.resolve_schema(schema, pascal_case(schema_name))
            self._resolved_component_types[schema_name] = resolved
            return resolved
        finally:
            self._resolving_refs.remove(schema_name)

    def _build_model(self, model_name: str, schema: dict[str, Any]) -> None:
        if model_name in self._state.model_definitions:
            return

        properties_raw = schema.get("properties")
        properties = properties_raw if isinstance(properties_raw, dict) else {}
        required_raw = schema.get("required")
        required_fields = set(required_raw) if isinstance(required_raw, list) else set()

        fields: list[ModelField] = []
        for property_name, property_schema in properties.items():
            if not isinstance(property_name, str):
                continue
            nested = property_schema if isinstance(property_schema, dict) else None
            annotation = self.resolve_schema(nested, f"{model_name}{pascal_case(property_name)}")
            is_required = property_name in required_fields
            if not is_required:
                annotation = add_optional(annotation, True)
            python_name, alias = python_field_name(property_name)
            if alias is not None:
                self._state.uses_field_alias = True
            fields.append(
                ModelField(
                    original_name=property_name,
                    python_name=python_name,
                    annotation=annotation,
                    required=is_required,
                    alias=alias,
                )
            )

        self._state.model_definitions[model_name] = ModelDefinition(name=model_name, fields=fields)
        self._state.model_order.append(model_name)

    @staticmethod
    def _is_mapping_schema(schema: dict[str, Any]) -> bool:
        if "additionalProperties" not in schema:
            return False
        properties = schema.get("properties")
        return not (isinstance(properties, dict) and properties)

    @staticmethod
    def _is_model_schema(schema: dict[str, Any]) -> bool:
        properties = schema.get("properties")
        has_additional = "additionalProperties" in schema
        if isinstance(properties, dict):
            if properties:
                return True
            return not has_additional
        return schema.get("type") == "object" and not has_additional


# ---------------------------------------------------------------------------
# Spec loading
# ---------------------------------------------------------------------------


def fetch_openapi() -> str:
    """Download the consolidated OpenAPI document, retrying on transient errors."""
    last_error: Exception | None = None
    for attempt in range(3):
        try:
            with urlopen(OPENAPI_URL, timeout=40) as response:  # noqa: S310 - trusted vendor docs
                return response.read().decode("utf-8")
        except Exception as error:  # noqa: BLE001 - retried, then re-raised with context
            last_error = error
            if attempt == 2:
                break
    assert last_error is not None
    raise RuntimeError(f"Failed to fetch {OPENAPI_URL}") from last_error


def refresh_vendored_spec() -> dict[str, Any]:
    """Re-download the spec, write it deterministically, and return it parsed."""
    spec = json.loads(fetch_openapi())
    SPEC_DIR.mkdir(parents=True, exist_ok=True)
    SPEC_PATH.write_text(
        json.dumps(spec, indent=2, sort_keys=True, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )
    return spec


def load_vendored_spec() -> dict[str, Any]:
    """Load the vendored spec snapshot from disk."""
    if not SPEC_PATH.exists():
        raise RuntimeError(
            f"Vendored spec not found at {SPEC_PATH}. Run with --refresh to download it."
        )
    parsed = json.loads(SPEC_PATH.read_text(encoding="utf-8"))
    if not isinstance(parsed, dict):
        raise RuntimeError(f"Vendored spec at {SPEC_PATH} is not a JSON object")
    return parsed


# ---------------------------------------------------------------------------
# Naming helpers
# ---------------------------------------------------------------------------


def snake_case(value: str) -> str:
    with_underscores = re.sub(r"[^0-9a-zA-Z]+", "_", value)
    with_boundaries = re.sub(r"(.)([A-Z][a-z]+)", r"\1_\2", with_underscores)
    normalized = re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", with_boundaries)
    normalized = re.sub(r"_+", "_", normalized).lower().strip("_")
    if not normalized:
        normalized = "param"
    if normalized[0].isdigit():
        normalized = f"p_{normalized}"
    if keyword.iskeyword(normalized):
        normalized = f"{normalized}_"
    return normalized


def pascal_case(value: str) -> str:
    return "".join(piece.capitalize() for piece in snake_case(value).split("_"))


def python_field_name(field_name: str) -> tuple[str, str | None]:
    if field_name.isidentifier() and not keyword.iskeyword(field_name):
        return field_name, None
    candidate = snake_case(field_name)
    if not candidate.isidentifier():
        candidate = f"field_{candidate}"
    if keyword.iskeyword(candidate):
        candidate = f"{candidate}_"
    return candidate, field_name


def deduplicate(values: list[Any]) -> list[Any]:
    seen: set[Any] = set()
    out: list[Any] = []
    for value in values:
        if value in seen:
            continue
        seen.add(value)
        out.append(value)
    return out


def annotation_has_none(annotation: str) -> bool:
    return bool(re.search(r"\bNone\b", annotation))


def add_optional(annotation: str, include_none: bool) -> str:
    if not include_none or annotation_has_none(annotation):
        return annotation
    return f"{annotation} | None"


def extract_enum_values(raw_enum: Any) -> list[Any]:
    if not isinstance(raw_enum, list):
        return []
    values: list[Any] = []
    for entry in raw_enum:
        if isinstance(entry, dict) and "value" in entry:
            values.append(entry["value"])
        elif isinstance(entry, (str, int, float, bool)) or entry is None:
            values.append(entry)
    return values


def primitive_schema_type(schema_type: Any) -> str:
    return {
        "string": "str",
        "integer": "int",
        "number": "float",
        "boolean": "bool",
        "object": "dict[str, Any]",
        "array": "list[Any]",
    }.get(schema_type, "Any")


def parameter_schema_to_annotation(
    schema: dict[str, Any] | None, components: dict[str, Any]
) -> str:
    return _parameter_schema_to_annotation(schema, components, set())


def _parameter_schema_to_annotation(
    schema: dict[str, Any] | None,
    components: dict[str, Any],
    seen_refs: set[str],
) -> str:
    if not isinstance(schema, dict):
        return "Any"

    ref = schema.get("$ref")
    if isinstance(ref, str):
        prefix = "#/components/schemas/"
        if ref.startswith(prefix):
            name = ref[len(prefix) :]
            if name in seen_refs:
                return "Any"
            referenced = components.get(name)
            if isinstance(referenced, dict):
                return _parameter_schema_to_annotation(referenced, components, seen_refs | {name})
        return "Any"

    nullable = schema.get("nullable") is True
    schema_type: Any = schema.get("type")
    if isinstance(schema_type, list):
        nullable = nullable or "null" in schema_type
        non_null = [entry for entry in schema_type if entry != "null"]
        schema_type = non_null[0] if len(non_null) == 1 else None

    one_of = schema.get("oneOf")
    if isinstance(one_of, list):
        options = [
            _parameter_schema_to_annotation(
                option if isinstance(option, dict) else None, components, seen_refs
            )
            for option in one_of
        ]
        annotation = " | ".join(deduplicate(options)) if options else "Any"
        return add_optional(annotation, nullable)

    enum_values = extract_enum_values(schema.get("enum"))
    if enum_values:
        filtered = [value for value in enum_values if value is not None]
        if len(filtered) != len(enum_values):
            nullable = True
        if filtered:
            literal = ", ".join(repr(value) for value in deduplicate(filtered))
            annotation = f"Literal[{literal}]"
        else:
            annotation = "Any"
        return add_optional(annotation, nullable)

    if schema_type == "array":
        item_schema = schema.get("items") if isinstance(schema.get("items"), dict) else None
        item_annotation = _parameter_schema_to_annotation(item_schema, components, seen_refs)
        return add_optional(f"Sequence[{item_annotation}]", nullable)

    if schema_type == "object" or "properties" in schema or "additionalProperties" in schema:
        return add_optional("Mapping[str, Any]", nullable)

    return add_optional(primitive_schema_type(schema_type), nullable)


# ---------------------------------------------------------------------------
# Operation parsing
# ---------------------------------------------------------------------------


def derive_group(path: str) -> str:
    first_segment = path.strip("/").split("/", 1)[0]
    return snake_case(first_segment or "misc")


def derive_docs_url(operation_id: str) -> str:
    return f"{DOCS_REFERENCE_BASE}/{snake_case(operation_id).replace('_', '-')}"


def resolve_parameters(
    raw_parameters: Any, components_parameters: dict[str, Any]
) -> list[dict[str, Any]]:
    """Inline ``$ref`` parameters and drop SDK-managed headers."""
    resolved: list[dict[str, Any]] = []
    if not isinstance(raw_parameters, list):
        return resolved
    for raw_param in raw_parameters:
        if not isinstance(raw_param, dict):
            continue
        ref = raw_param.get("$ref")
        if isinstance(ref, str):
            prefix = "#/components/parameters/"
            if ref.startswith(prefix):
                referenced = components_parameters.get(ref[len(prefix) :])
                if isinstance(referenced, dict):
                    resolved.append(referenced)
            continue
        resolved.append(raw_param)
    return resolved


def pick_success_response(
    responses: dict[str, Any],
) -> tuple[str | None, dict[str, Any] | None]:
    candidates: list[tuple[int, str, dict[str, Any]]] = []
    for status_code, response in responses.items():
        if not str(status_code).startswith("2") or not isinstance(response, dict):
            continue
        try:
            numeric = int(str(status_code))
        except ValueError:
            numeric = 999
        candidates.append((numeric, str(status_code), response))

    if not candidates:
        return None, None
    candidates.sort(key=lambda item: item[0])

    for _, status_code, response in candidates:
        content = response.get("content")
        if not isinstance(content, dict):
            continue
        json_content = content.get("application/json")
        if isinstance(json_content, dict) and isinstance(json_content.get("schema"), dict):
            return status_code, json_content["schema"]

    return candidates[0][1], None


def parse_operation(
    *,
    method: str,
    path: str,
    method_spec: dict[str, Any],
    components_schemas: dict[str, Any],
    components_parameters: dict[str, Any],
    resolver: SchemaResolver,
    state: ModelBuildState,
) -> Operation:
    raw_operation_id = method_spec.get("operationId")
    if not isinstance(raw_operation_id, str):
        raise RuntimeError(f"Missing operationId for {method} {path}")

    operation_id = snake_case(raw_operation_id)
    operation_prefix = pascal_case(operation_id)

    used_python_names: set[str] = set()
    params: list[Param] = []
    for raw_param in resolve_parameters(method_spec.get("parameters"), components_parameters):
        name = raw_param.get("name")
        location = raw_param.get("in")
        if not isinstance(name, str) or location not in {"path", "query", "header"}:
            continue
        if location == "header" and name.lower() in SDK_MANAGED_HEADERS:
            continue

        base_python_name = snake_case(name)
        python_name = base_python_name
        index = 2
        while python_name in used_python_names:
            python_name = f"{base_python_name}_{index}"
            index += 1
        used_python_names.add(python_name)

        annotation = parameter_schema_to_annotation(raw_param.get("schema"), components_schemas)
        params.append(
            Param(
                name=name,
                python_name=python_name,
                location=location,
                required=bool(raw_param.get("required", False)),
                annotation=annotation,
            )
        )

    request_body = method_spec.get("requestBody")
    request_body_required = False
    request_body_content_types: tuple[str, ...] = ()
    if isinstance(request_body, dict):
        request_body_required = bool(request_body.get("required", False))
        content = request_body.get("content")
        if isinstance(content, dict):
            request_body_content_types = tuple(content.keys())

    responses = method_spec.get("responses") if isinstance(method_spec.get("responses"), dict) else {}
    status_code, response_schema = pick_success_response(responses)

    if status_code == "204":
        response_type_name = "None"
    elif isinstance(response_schema, dict):
        response_type_expr = resolver.resolve_schema(
            response_schema, f"{operation_prefix}ResponsePayload"
        )
        response_type_name = state.reserve_type_name(f"{operation_prefix}Response")
        state.response_type_aliases[response_type_name] = response_type_expr
        state.response_alias_order.append(response_type_name)
        state.operation_response_aliases[operation_id] = response_type_name
    else:
        response_type_name = "None"

    summary = method_spec.get("summary")
    description = method_spec.get("description")
    return Operation(
        operation_id=operation_id,
        group=derive_group(path),
        method=method.upper(),
        path=path,
        summary=summary if isinstance(summary, str) and summary else operation_id,
        description=description if isinstance(description, str) else "",
        docs_url=derive_docs_url(operation_id),
        parameters=params,
        request_body_required=request_body_required,
        request_body_content_types=request_body_content_types,
        response_type_name=response_type_name,
    )


def collect_operations(spec: dict[str, Any]) -> tuple[list[Operation], ModelBuildState]:
    paths = spec.get("paths")
    if not isinstance(paths, dict):
        raise RuntimeError("Spec is missing a 'paths' object")

    components = spec.get("components") if isinstance(spec.get("components"), dict) else {}
    components_schemas = (
        dict(components["schemas"]) if isinstance(components.get("schemas"), dict) else {}
    )
    components_parameters = (
        dict(components["parameters"]) if isinstance(components.get("parameters"), dict) else {}
    )

    state = ModelBuildState(
        used_type_names=set(),
        model_definitions={},
        model_order=[],
        response_type_aliases={},
        response_alias_order=[],
        operation_response_aliases={},
        requires_literal=False,
        uses_field_alias=False,
    )
    resolver = SchemaResolver(state=state, components=components_schemas)

    operations: list[Operation] = []
    seen_operation_ids: set[str] = set()
    for path, path_item in paths.items():
        if not isinstance(path_item, dict):
            continue
        for method, method_spec in path_item.items():
            if method.lower() not in HTTP_METHODS or not isinstance(method_spec, dict):
                continue
            operation = parse_operation(
                method=method,
                path=path,
                method_spec=method_spec,
                components_schemas=components_schemas,
                components_parameters=components_parameters,
                resolver=resolver,
                state=state,
            )
            if operation.operation_id in seen_operation_ids:
                raise RuntimeError(f"Duplicate operation_id detected: {operation.operation_id}")
            seen_operation_ids.add(operation.operation_id)
            operations.append(operation)

    operations.sort(key=lambda item: (item.group, item.operation_id))
    return operations, state


# ---------------------------------------------------------------------------
# Renderers
# ---------------------------------------------------------------------------


def render_operations_module(operations: list[Operation]) -> str:
    lines: list[str] = [
        "from __future__ import annotations",
        "",
        "from fanvue_sdk.types import OperationSpec, ParameterSpec",
        "",
        "OPERATIONS: dict[str, OperationSpec] = {",
    ]
    for operation in operations:
        lines.append(f"    {operation.operation_id!r}: OperationSpec(")
        lines.append(f"        operation_id={operation.operation_id!r},")
        lines.append(f"        group={operation.group!r},")
        lines.append(f"        method={operation.method!r},")
        lines.append(f"        path={operation.path!r},")
        lines.append(f"        summary={operation.summary!r},")
        lines.append(f"        description={operation.description!r},")
        lines.append(f"        docs_url={operation.docs_url!r},")
        if operation.parameters:
            lines.append("        parameters=(")
            for param in operation.parameters:
                lines.append(
                    "            ParameterSpec("
                    f"name={param.name!r}, location={param.location!r}, "
                    f"required={param.required}),"
                )
            lines.append("        ),")
        else:
            lines.append("        parameters=(),")
        if operation.request_body_content_types:
            types_literal = ", ".join(repr(ct) for ct in operation.request_body_content_types)
            if len(operation.request_body_content_types) == 1:
                types_literal = f"{types_literal},"
            lines.append(f"        request_body_content_types=({types_literal}),")
        else:
            lines.append("        request_body_content_types=(),")
        lines.append(f"        request_body_required={operation.request_body_required},")
        lines.append("    ),")
    lines.append("}")
    lines.append("")
    lines.append("OPERATION_IDS: tuple[str, ...] = tuple(OPERATIONS.keys())")
    lines.append("")
    return "\n".join(lines)


def render_models_module(state: ModelBuildState) -> str:
    lines: list[str] = ["from __future__ import annotations", ""]

    typing_imports = ["Any", "TypeAlias"]
    if state.requires_literal:
        typing_imports.append("Literal")
    lines.append(f"from typing import {', '.join(sorted(typing_imports))}")
    lines.append("")

    pydantic_imports = ["BaseModel", "ConfigDict", "TypeAdapter"]
    if state.uses_field_alias:
        pydantic_imports.append("Field")
    lines.append(f"from pydantic import {', '.join(sorted(pydantic_imports))}")
    lines.append("")
    lines.append("from fanvue_sdk.types import ResponseData")
    lines.append("")
    lines.append("")
    lines.append("class FanvueModel(BaseModel):")
    lines.append('    """Base model for generated Fanvue response payloads."""')
    lines.append("")
    lines.append("    model_config = ConfigDict(extra='allow', populate_by_name=True)")

    for model_name in state.model_order:
        model = state.model_definitions[model_name]
        lines.append("")
        lines.append("")
        lines.append(f"class {model.name}(FanvueModel):")
        if not model.fields:
            lines.append("    pass")
            continue
        for field in model.fields:
            assignment = f"{field.python_name}: {field.annotation}"
            if field.required:
                if field.alias is not None:
                    assignment += f" = Field(alias={field.original_name!r})"
            elif field.alias is not None:
                assignment += f" = Field(default=None, alias={field.original_name!r})"
            else:
                assignment += " = None"
            lines.append(f"    {assignment}")

    lines.append("")
    lines.append("")
    for alias_name in state.response_alias_order:
        lines.append(f"{alias_name}: TypeAlias = {state.response_type_aliases[alias_name]}")

    lines.append("")
    lines.append("")
    lines.append("OPERATION_RESPONSE_ADAPTERS: dict[str, TypeAdapter[Any]] = {")
    for operation_id in sorted(state.operation_response_aliases):
        alias_name = state.operation_response_aliases[operation_id]
        lines.append(f"    {operation_id!r}: TypeAdapter({alias_name}),")
    lines.append("}")

    lines.append("")
    lines.append("")
    lines.append("def parse_operation_response(operation_id: str, payload: ResponseData) -> Any:")
    lines.append('    """Validate a raw payload into its typed model for ``operation_id``."""')
    lines.append("    if payload is None or isinstance(payload, bytes):")
    lines.append("        return payload")
    lines.append("")
    lines.append("    adapter = OPERATION_RESPONSE_ADAPTERS.get(operation_id)")
    lines.append("    if adapter is None:")
    lines.append("        return payload")
    lines.append("")
    lines.append("    return adapter.validate_python(payload)")
    return "\n".join(lines)


def ordered_path_params(parameters: list[Param], path: str) -> list[Param]:
    path_param_names = PATH_PARAMETER_PATTERN.findall(path)
    path_param_map = {p.name: p for p in parameters if p.location == "path"}
    return [path_param_map[name] for name in path_param_names if name in path_param_map]


def render_resource_module(group: str, operations: list[Operation]) -> str:
    class_name = f"{pascal_case(group)}Resource"

    model_return_types = sorted(
        {op.response_type_name for op in operations if op.response_type_name not in {"None", "Any"}},
        key=str.lower,
    )
    parameter_annotations = [p.annotation for op in operations for p in op.parameters]
    has_any_body = any(op.request_body_content_types for op in operations)

    uses_mapping = any("Mapping[" in a for a in parameter_annotations) or has_any_body
    uses_sequence = any("Sequence[" in a for a in parameter_annotations)
    uses_any = any("Any" in a for a in parameter_annotations) or has_any_body
    uses_literal = any("Literal[" in a for a in parameter_annotations)

    lines: list[str] = ["from __future__ import annotations", ""]

    collection_imports: list[str] = []
    if uses_mapping:
        collection_imports.append("Mapping")
    if uses_sequence:
        collection_imports.append("Sequence")
    if collection_imports:
        lines.append(f"from collections.abc import {', '.join(collection_imports)}")

    typing_imports: list[str] = []
    if uses_any:
        typing_imports.append("Any")
    if uses_literal:
        typing_imports.append("Literal")
    typing_imports.append("cast")
    lines.append(f"from typing import {', '.join(sorted(set(typing_imports)))}")

    if collection_imports or typing_imports:
        lines.append("")

    if model_return_types:
        lines.append("from fanvue_sdk.models import (")
        for model_name in model_return_types:
            lines.append(f"    {model_name},")
        lines.append(")")
    lines.append("from fanvue_sdk.resources.base import BaseResource")
    lines.append("")
    lines.append("")
    lines.append(f"class {class_name}(BaseResource):")
    lines.append(f'    """{class_name} endpoints."""')

    for operation in operations:
        path_params = ordered_path_params(operation.parameters, operation.path)
        query_params = [p for p in operation.parameters if p.location == "query"]
        header_params = [p for p in operation.parameters if p.location == "header"]
        required_query = [p for p in query_params if p.required]
        optional_query = [p for p in query_params if not p.required]
        required_headers = [p for p in header_params if p.required]
        optional_headers = [p for p in header_params if not p.required]
        has_body = bool(operation.request_body_content_types)

        signature_parts: list[str] = ["self"]
        for param in path_params:
            signature_parts.append(f"{param.python_name}: {param.annotation}")

        has_keyword_only = bool(
            required_query or optional_query or required_headers or optional_headers or has_body
        )
        if has_keyword_only:
            signature_parts.append("*")
        for param in required_query:
            signature_parts.append(f"{param.python_name}: {param.annotation}")
        for param in optional_query:
            signature_parts.append(
                f"{param.python_name}: {add_optional(param.annotation, True)} = None"
            )
        for param in required_headers:
            signature_parts.append(f"{param.python_name}: {param.annotation}")
        for param in optional_headers:
            signature_parts.append(
                f"{param.python_name}: {add_optional(param.annotation, True)} = None"
            )
        if has_body:
            if operation.request_body_required:
                signature_parts.append("body: Mapping[str, Any]")
            else:
                signature_parts.append("body: Mapping[str, Any] | None = None")

        signature = ", ".join(signature_parts)
        return_type = operation.response_type_name

        lines.append("")
        lines.append(f"    async def {operation.operation_id}({signature}) -> {return_type}:")
        lines.append('        """')
        lines.append(f"        {operation.summary}")
        lines.append("")
        lines.append(f"        `{operation.method} {operation.path}`")
        lines.append(f"        Docs: {operation.docs_url}")
        lines.append('        """')
        if return_type == "None":
            lines.append("        await self._client._call_operation(")
        else:
            lines.append(
                f"        return cast({return_type}, await self._client._call_operation("
            )
        lines.append(f"            operation_id={operation.operation_id!r},")
        if path_params:
            lines.append("            path_params={")
            for param in path_params:
                lines.append(f"                {param.name!r}: {param.python_name},")
            lines.append("            },")
        if query_params:
            lines.append("            query_params={")
            for param in query_params:
                lines.append(f"                {param.name!r}: {param.python_name},")
            lines.append("            },")
        if header_params:
            lines.append("            header_params={")
            for param in header_params:
                lines.append(f"                {param.name!r}: {param.python_name},")
            lines.append("            },")
        if has_body:
            lines.append("            body=body,")
        if return_type == "None":
            lines.append("        )")
            lines.append("        return None")
        else:
            lines.append("        ))")

    lines.append("")
    return "\n".join(lines)


def render_resources_init(groups: list[str]) -> str:
    class_names = [f"{pascal_case(group)}Resource" for group in groups]
    lines: list[str] = ["from __future__ import annotations", ""]
    for group, class_name in zip(groups, class_names, strict=True):
        lines.append(f"from fanvue_sdk.resources.{group} import {class_name}")
    lines.append("")
    lines.append("RESOURCE_FACTORIES = {")
    for group, class_name in zip(groups, class_names, strict=True):
        lines.append(f"    {group!r}: {class_name},")
    lines.append("}")
    lines.append("")
    lines.append("__all__ = [")
    for name in [*class_names, "RESOURCE_FACTORIES"]:
        lines.append(f"    {name!r},")
    lines.append("]")
    lines.append("")
    return "\n".join(lines)


def render_endpoints_markdown(operations: list[Operation]) -> str:
    lines = [
        "| Group | Operation | Method | Path | Response Type |",
        "| --- | --- | --- | --- | --- |",
    ]
    for operation in operations:
        lines.append(
            f"| `{operation.group}` | `{operation.operation_id}` | `{operation.method}` "
            f"| `{operation.path}` | `{operation.response_type_name}` |"
        )
    return "\n".join(lines) + "\n"


def write(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    body = content if content.endswith("\n") else content + "\n"
    path.write_text(
        f"# This file is auto-generated by scripts/generate_sdk.py\n\n{body}",
        encoding="utf-8",
    )


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------


def generate(spec: dict[str, Any]) -> int:
    operations, state = collect_operations(spec)

    grouped: dict[str, list[Operation]] = defaultdict(list)
    for operation in operations:
        grouped[operation.group].append(operation)
    for group_ops in grouped.values():
        group_ops.sort(key=lambda item: item.operation_id)

    write(PACKAGE_DIR / "_operations.py", render_operations_module(operations))
    write(PACKAGE_DIR / "models.py", render_models_module(state))

    groups = sorted(grouped)
    for group in groups:
        write(RESOURCES_DIR / f"{group}.py", render_resource_module(group, grouped[group]))
    write(RESOURCES_DIR / "__init__.py", render_resources_init(groups))

    (ROOT / "docs-endpoints.md").write_text(
        render_endpoints_markdown(operations), encoding="utf-8"
    )

    print(
        f"Generated {len(operations)} operations across {len(groups)} resource groups, "
        f"{len(state.model_order)} models, {len(state.response_alias_order)} response aliases."
    )
    return len(operations)


def main(argv: list[str] | None = None) -> None:
    parser = argparse.ArgumentParser(description="Generate the Fanvue SDK from the OpenAPI spec.")
    parser.add_argument(
        "--refresh",
        action="store_true",
        help="Re-download the OpenAPI spec and overwrite the vendored snapshot before generating.",
    )
    args = parser.parse_args(argv)

    spec = refresh_vendored_spec() if args.refresh else load_vendored_spec()
    generate(spec)


if __name__ == "__main__":
    sys.exit(main())
