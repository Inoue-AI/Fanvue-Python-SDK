#!/usr/bin/env python3
from __future__ import annotations

import keyword
import re
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Any
from urllib.request import urlopen

import yaml

ROOT = Path(__file__).resolve().parents[1]
PACKAGE_DIR = ROOT / "fanvue_sdk"
RESOURCES_DIR = PACKAGE_DIR / "resources"

INDEX_URL = "https://api.fanvue.com/docs/llms.txt"
API_REFERENCE_LINE = re.compile(
    r"^- Reference(?: > (?P<group>[^\[]+))? \[(?P<title>[^\]]+)\]\((?P<url>[^)]+)\)"
)
OPENAPI_BLOCK = re.compile(
    r"## OpenAPI Specification\s*```yaml\n(?P<yaml>.*?)\n```", re.DOTALL
)
PATH_PARAMETER_PATTERN = re.compile(r"\{([^}]+)\}")
HTTP_METHODS = {"get", "post", "put", "patch", "delete", "options", "head"}


@dataclass(slots=True)
class Param:
    name: str
    python_name: str
    location: str
    required: bool
    annotation: str


@dataclass(slots=True)
class Operation:
    operation_id: str
    group: str
    title: str
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
    original_name: str
    python_name: str
    annotation: str
    required: bool
    alias: str | None


@dataclass(slots=True)
class ModelDefinition:
    name: str
    fields: list[ModelField]


@dataclass(slots=True)
class ModelBuildState:
    used_type_names: set[str]
    model_definitions: dict[str, ModelDefinition]
    model_order: list[str]
    response_type_aliases: dict[str, str]
    response_alias_order: list[str]
    operation_response_aliases: dict[str, str]
    requires_literal: bool
    uses_field_alias: bool

    def reserve_type_name(self, base_name: str) -> str:
        normalized = pascal_case(base_name)
        if not normalized:
            normalized = "GeneratedType"

        candidate = normalized
        index = 2
        while candidate in self.used_type_names:
            candidate = f"{normalized}{index}"
            index += 1

        self.used_type_names.add(candidate)
        return candidate


class SchemaResolver:
    def __init__(
        self,
        *,
        state: ModelBuildState,
        operation_prefix: str,
        components: dict[str, Any],
    ) -> None:
        self._state = state
        self._operation_prefix = operation_prefix
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

        one_of = schema.get("oneOf")
        if isinstance(one_of, list):
            return self._resolve_union(one_of, hint)

        any_of = schema.get("anyOf")
        if isinstance(any_of, list):
            return self._resolve_union(any_of, hint)

        all_of = schema.get("allOf")
        if isinstance(all_of, list):
            return self._resolve_union(all_of, hint)

        nullable = False
        schema_type: Any = schema.get("type")
        if isinstance(schema_type, list):
            nullable = "null" in schema_type
            non_null_types = [entry for entry in schema_type if entry != "null"]
            schema_type = non_null_types[0] if len(non_null_types) == 1 else None
        elif schema_type == "null":
            return "None"

        enum_values = extract_enum_values(schema.get("enum"))
        if enum_values:
            filtered_values = [value for value in enum_values if value is not None]
            if len(filtered_values) != len(enum_values):
                nullable = True

            if filtered_values:
                self._state.requires_literal = True
                literal_values = ", ".join(repr(value) for value in deduplicate(filtered_values))
                annotation = f"Literal[{literal_values}]"
            else:
                annotation = "Any"

            return add_optional(annotation, nullable)

        if schema_type == "array":
            item_schema = schema.get("items") if isinstance(schema.get("items"), dict) else None
            item_type = self.resolve_schema(item_schema, f"{hint}Item")
            return add_optional(f"list[{item_type}]", nullable)

        if self._is_mapping_schema(schema):
            additional = schema.get("additionalProperties")
            if additional in (None, True, False):
                value_type = "Any"
            elif isinstance(additional, dict):
                value_type = self.resolve_schema(additional, f"{hint}Value")
            else:
                value_type = "Any"

            return add_optional(f"dict[str, {value_type}]", nullable)

        if self._is_model_schema(schema):
            model_name = self._state.reserve_type_name(hint)
            self._build_model(model_name, schema)
            return add_optional(model_name, nullable)

        primitive_type = primitive_schema_type(schema_type)
        return add_optional(primitive_type, nullable)

    def _resolve_union(self, variants: list[Any], hint: str) -> str:
        options: list[str] = []
        for index, option in enumerate(variants, start=1):
            option_schema = option if isinstance(option, dict) else None
            options.append(self.resolve_schema(option_schema, f"{hint}Option{index}"))

        compact_options = deduplicate(options)
        if not compact_options:
            return "Any"

        return " | ".join(compact_options)

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
                    model_name = self._state.reserve_type_name(
                        f"{self._operation_prefix}{pascal_case(schema_name)}"
                    )
                    self._component_model_names[schema_name] = model_name

                self._resolved_component_types[schema_name] = model_name
                self._build_model(model_name, schema)
                return model_name

            resolved = self.resolve_schema(
                schema,
                f"{self._operation_prefix}{pascal_case(schema_name)}",
            )
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

            nested_schema = property_schema if isinstance(property_schema, dict) else None
            annotation = self.resolve_schema(
                nested_schema,
                f"{model_name}{pascal_case(property_name)}",
            )

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

        schema_type = schema.get("type")
        return schema_type == "object" and not has_additional


def fetch_text(url: str) -> str:
    last_error: Exception | None = None
    for attempt in range(3):
        try:
            with urlopen(url, timeout=20) as response:  # noqa: S310 - trusted docs source for generation
                return response.read().decode("utf-8")
        except Exception as error:  # noqa: BLE001 - retries then fail with context
            last_error = error
            if attempt == 2:
                break

    assert last_error is not None
    raise RuntimeError(f"Failed to fetch {url}") from last_error


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
    deduplicated: list[Any] = []
    for value in values:
        if value in seen:
            continue
        seen.add(value)
        deduplicated.append(value)
    return deduplicated


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
    if schema_type == "string":
        return "str"
    if schema_type == "integer":
        return "int"
    if schema_type == "number":
        return "float"
    if schema_type == "boolean":
        return "bool"
    if schema_type == "object":
        return "dict[str, Any]"
    if schema_type == "array":
        return "list[Any]"
    return "Any"


def parameter_schema_to_annotation(schema: dict[str, Any] | None, components: dict[str, Any]) -> str:
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

    nullable = False
    schema_type: Any = schema.get("type")
    if isinstance(schema_type, list):
        nullable = "null" in schema_type
        non_null_types = [entry for entry in schema_type if entry != "null"]
        schema_type = non_null_types[0] if len(non_null_types) == 1 else None

    one_of = schema.get("oneOf")
    if isinstance(one_of, list):
        options = [
            _parameter_schema_to_annotation(
                option if isinstance(option, dict) else None,
                components,
                seen_refs,
            )
            for option in one_of
        ]
        annotation = " | ".join(deduplicate(options)) if options else "Any"
        return add_optional(annotation, nullable)

    enum_values = extract_enum_values(schema.get("enum"))
    if enum_values:
        filtered_values = [value for value in enum_values if value is not None]
        if len(filtered_values) != len(enum_values):
            nullable = True

        if filtered_values:
            literal_values = ", ".join(repr(value) for value in deduplicate(filtered_values))
            annotation = f"Literal[{literal_values}]"
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


def parse_api_index(text: str) -> list[tuple[str | None, str, str]]:
    entries: list[tuple[str | None, str, str]] = []

    for raw_line in text.splitlines():
        line = raw_line.strip()
        match = API_REFERENCE_LINE.match(line)
        if not match:
            continue

        group = match.group("group")
        title = match.group("title").strip()
        url = match.group("url").strip()
        entries.append((group.strip() if group else None, title, url))

    return entries


def derive_group(group: str | None, path: str) -> str:
    if group:
        return snake_case(group)

    first_segment = path.strip("/").split("/", 1)[0]
    return snake_case(first_segment or "misc")


def extract_openapi(text: str, *, source_url: str) -> dict[str, Any]:
    block_match = OPENAPI_BLOCK.search(text)
    if block_match is None:
        raise RuntimeError(f"Could not find OpenAPI block in {source_url}")

    parsed = yaml.safe_load(block_match.group("yaml"))
    if not isinstance(parsed, dict):
        raise RuntimeError(f"Unexpected OpenAPI structure in {source_url}")

    return parsed


def pick_success_response(
    responses: dict[str, Any],
) -> tuple[str | None, dict[str, Any] | None, dict[str, Any] | None]:
    success_candidates: list[tuple[int, str, dict[str, Any]]] = []
    for status_code, response in responses.items():
        if not str(status_code).startswith("2"):
            continue
        if not isinstance(response, dict):
            continue

        try:
            numeric_code = int(str(status_code))
        except ValueError:
            numeric_code = 999

        success_candidates.append((numeric_code, str(status_code), response))

    if not success_candidates:
        return None, None, None

    success_candidates.sort(key=lambda item: item[0])

    for _, status_code, response in success_candidates:
        content = response.get("content")
        if not isinstance(content, dict):
            continue
        json_content = content.get("application/json")
        if not isinstance(json_content, dict):
            continue
        schema = json_content.get("schema")
        if isinstance(schema, dict):
            return status_code, response, schema

    _, status_code, response = success_candidates[0]
    return status_code, response, None


def parse_operation(
    group: str | None,
    title: str,
    docs_url: str,
    page: str,
    state: ModelBuildState,
) -> Operation:
    specification = extract_openapi(page, source_url=docs_url)
    paths = specification.get("paths")
    if not isinstance(paths, dict) or len(paths) != 1:
        raise RuntimeError(f"Expected exactly one path in {docs_url}")

    path, path_item = next(iter(paths.items()))
    if not isinstance(path_item, dict):
        raise RuntimeError(f"Malformed OpenAPI path item in {docs_url}")

    method: str | None = None
    method_spec: dict[str, Any] | None = None
    for key, value in path_item.items():
        if key.lower() not in HTTP_METHODS:
            continue
        method = key.upper()
        if isinstance(value, dict):
            method_spec = value
            break

    if method is None or method_spec is None:
        raise RuntimeError(f"No HTTP method found for {docs_url}")

    raw_operation_id = method_spec.get("operationId")
    if not isinstance(raw_operation_id, str):
        raise RuntimeError(f"Missing operationId for {docs_url}")

    operation_id = snake_case(raw_operation_id.replace("-", "_"))
    operation_prefix = pascal_case(operation_id)

    components_raw = specification.get("components")
    components_schemas: dict[str, Any]
    if isinstance(components_raw, dict) and isinstance(components_raw.get("schemas"), dict):
        components_schemas = dict(components_raw["schemas"])
    else:
        components_schemas = {}

    raw_parameters = method_spec.get("parameters", [])
    if not isinstance(raw_parameters, list):
        raise RuntimeError(f"Unexpected parameters type for {docs_url}")

    used_python_names: set[str] = set()
    params: list[Param] = []

    for raw_param in raw_parameters:
        if not isinstance(raw_param, dict):
            continue

        name = raw_param.get("name")
        location = raw_param.get("in")

        if not isinstance(name, str) or location not in {"path", "query", "header"}:
            continue
        if name in {"Authorization", "X-Fanvue-API-Version"}:
            continue

        base_python_name = snake_case(name)
        python_name = base_python_name
        index = 2
        while python_name in used_python_names:
            python_name = f"{base_python_name}_{index}"
            index += 1

        used_python_names.add(python_name)

        annotation = parameter_schema_to_annotation(raw_param.get("schema"), components_schemas)
        required = bool(raw_param.get("required", False))

        params.append(
            Param(
                name=name,
                python_name=python_name,
                location=location,
                required=required,
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
    status_code, _response_spec, response_schema = pick_success_response(responses)

    if status_code == "204":
        response_type_name = "None"
    elif isinstance(response_schema, dict):
        resolver = SchemaResolver(
            state=state,
            operation_prefix=operation_prefix,
            components=components_schemas,
        )
        response_type_expr = resolver.resolve_schema(
            response_schema,
            f"{operation_prefix}ResponsePayload",
        )
        response_type_name = state.reserve_type_name(f"{operation_prefix}Response")
        state.response_type_aliases[response_type_name] = response_type_expr
        state.response_alias_order.append(response_type_name)
        state.operation_response_aliases[operation_id] = response_type_name
    else:
        response_type_name = "Any"

    summary = method_spec.get("summary")
    description = method_spec.get("description")

    return Operation(
        operation_id=operation_id,
        group=derive_group(group, path),
        title=title,
        method=method,
        path=path,
        summary=summary if isinstance(summary, str) else title,
        description=description if isinstance(description, str) else "",
        docs_url=docs_url.replace(".mdx", ""),
        parameters=params,
        request_body_required=request_body_required,
        request_body_content_types=request_body_content_types,
        response_type_name=response_type_name,
    )


def render_operations_module(operations: list[Operation]) -> str:
    lines: list[str] = []
    lines.append("from __future__ import annotations")
    lines.append("")
    lines.append("from fanvue_sdk.types import OperationSpec, ParameterSpec")
    lines.append("")
    lines.append("OPERATIONS: dict[str, OperationSpec] = {")

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
                    f"name={param.name!r}, location={param.location!r}, required={param.required}),"
                )
            lines.append("        ),")
        else:
            lines.append("        parameters=(),")

        if operation.request_body_content_types:
            types_literal = ", ".join(
                repr(content_type) for content_type in operation.request_body_content_types
            )
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
    lines: list[str] = []
    lines.append("from __future__ import annotations")
    lines.append("")

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
    lines.append("    \"\"\"Base model for generated Fanvue response payloads.\"\"\"")
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
            else:
                if field.alias is not None:
                    assignment += f" = Field(default=None, alias={field.original_name!r})"
                else:
                    assignment += " = None"
            lines.append(f"    {assignment}")

    lines.append("")
    lines.append("")
    for alias_name in state.response_alias_order:
        alias_expression = state.response_type_aliases[alias_name]
        lines.append(f"{alias_name}: TypeAlias = {alias_expression}")

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
    path_param_map = {
        parameter.name: parameter for parameter in parameters if parameter.location == "path"
    }

    return [path_param_map[name] for name in path_param_names if name in path_param_map]


def render_resource_module(group: str, operations: list[Operation]) -> str:
    class_name = f"{pascal_case(group)}Resource"

    model_return_types = sorted(
        {
            operation.response_type_name
            for operation in operations
            if operation.response_type_name not in {"None", "Any"}
        },
        key=str.lower,
    )

    parameter_annotations = [
        parameter.annotation for operation in operations for parameter in operation.parameters
    ]

    uses_mapping = any("Mapping[" in annotation for annotation in parameter_annotations) or any(
        bool(operation.request_body_content_types) for operation in operations
    )
    uses_sequence = any("Sequence[" in annotation for annotation in parameter_annotations)

    uses_any = any("Any" in annotation for annotation in parameter_annotations) or any(
        bool(operation.request_body_content_types) for operation in operations
    )
    uses_literal = any("Literal[" in annotation for annotation in parameter_annotations)

    lines: list[str] = []
    lines.append("from __future__ import annotations")
    lines.append("")

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
    if typing_imports:
        sorted_typing_imports = sorted(set(typing_imports))
        lines.append(f"from typing import {', '.join(sorted_typing_imports)}")

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
    lines.append(f"    \"\"\"{class_name} endpoints.\"\"\"")

    for operation in operations:
        path_params = ordered_path_params(operation.parameters, operation.path)
        query_params = [param for param in operation.parameters if param.location == "query"]
        header_params = [param for param in operation.parameters if param.location == "header"]

        required_query = [param for param in query_params if param.required]
        optional_query = [param for param in query_params if not param.required]
        required_headers = [param for param in header_params if param.required]
        optional_headers = [param for param in header_params if not param.required]

        has_body = bool(operation.request_body_content_types)

        signature_parts: list[str] = ["self"]
        for param in path_params:
            signature_parts.append(f"{param.python_name}: {param.annotation}")

        has_keyword_only = bool(
            required_query
            or optional_query
            or required_headers
            or optional_headers
            or has_body
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

        lines.append("")
        lines.append(
            f"    async def {operation.operation_id}({signature}) -> {operation.response_type_name}:"
        )
        lines.append("        \"\"\"")
        lines.append(f"        {operation.summary}")
        lines.append("")
        lines.append(f"        `{operation.method} {operation.path}`")
        lines.append(f"        Docs: {operation.docs_url}")
        lines.append("        \"\"\"")
        if operation.response_type_name == "None":
            lines.append("        await self._client._call_operation(")
        else:
            lines.append(
                f"        return cast({operation.response_type_name}, await self._client._call_operation("
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

        if operation.response_type_name == "None":
            lines.append("        )")
            lines.append("        return None")
        else:
            lines.append("        ))")

    lines.append("")
    return "\n".join(lines)


def render_resources_init(groups: list[str]) -> str:
    lines: list[str] = []
    lines.append("from __future__ import annotations")
    lines.append("")

    class_names = [f"{pascal_case(group)}Resource" for group in groups]
    for group, class_name in zip(groups, class_names, strict=True):
        lines.append(f"from fanvue_sdk.resources.{group} import {class_name}")

    lines.append("")
    lines.append("RESOURCE_FACTORIES = {")
    for group, class_name in zip(groups, class_names, strict=True):
        lines.append(f"    {group!r}: {class_name},")
    lines.append("}")
    lines.append("")

    exported = [*class_names, "RESOURCE_FACTORIES"]
    lines.append("__all__ = [")
    for name in exported:
        lines.append(f"    {name!r},")
    lines.append("]")
    lines.append("")

    return "\n".join(lines)


def render_readme_table(operations: list[Operation]) -> str:
    lines: list[str] = []
    lines.append("| Group | Operation | Method | Path | Response Type |")
    lines.append("| --- | --- | --- | --- | --- |")

    for operation in operations:
        lines.append(
            "| "
            f"`{operation.group}` | "
            f"`{operation.operation_id}` | "
            f"`{operation.method}` | "
            f"`{operation.path}` | "
            f"`{operation.response_type_name}` "
            "|"
        )

    return "\n".join(lines)


def write(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(f"# This file is auto-generated by scripts/generate_sdk.py\n\n{content}")


def main() -> None:
    index_text = fetch_text(INDEX_URL)
    index_entries = parse_api_index(index_text)

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

    operations: list[Operation] = []
    seen_operation_ids: set[str] = set()

    for group, title, docs_url in index_entries:
        page_text = fetch_text(docs_url)
        operation = parse_operation(group, title, docs_url, page_text, state)
        if operation.operation_id in seen_operation_ids:
            raise RuntimeError(f"Duplicate operation_id detected: {operation.operation_id}")

        seen_operation_ids.add(operation.operation_id)
        operations.append(operation)

    operations.sort(key=lambda item: (item.group, item.operation_id))

    grouped_operations: dict[str, list[Operation]] = defaultdict(list)
    for operation in operations:
        grouped_operations[operation.group].append(operation)

    for operation_group in grouped_operations.values():
        operation_group.sort(key=lambda item: item.operation_id)

    operations_module = render_operations_module(operations)
    write(PACKAGE_DIR / "_operations.py", operations_module)

    models_module = render_models_module(state)
    write(PACKAGE_DIR / "models.py", models_module)

    groups = sorted(grouped_operations)
    for group in groups:
        resource_module = render_resource_module(group, grouped_operations[group])
        write(RESOURCES_DIR / f"{group}.py", resource_module)

    resources_init = render_resources_init(groups)
    write(RESOURCES_DIR / "__init__.py", resources_init)

    endpoint_markdown = render_readme_table(operations)
    (ROOT / "docs-endpoints.md").write_text(endpoint_markdown)

    print(
        f"Generated {len(operations)} operations, "
        f"{len(state.model_order)} models, and "
        f"{len(state.response_alias_order)} response aliases"
    )


if __name__ == "__main__":
    main()
