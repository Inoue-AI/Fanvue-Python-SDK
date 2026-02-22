from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Literal, TypeAlias

JSONPrimitive: TypeAlias = str | int | float | bool | None
JSONValue: TypeAlias = JSONPrimitive | dict[str, "JSONValue"] | list["JSONValue"]
ResponseData: TypeAlias = JSONValue | bytes | None


@dataclass(frozen=True, slots=True)
class ParameterSpec:
    """Describes a single OpenAPI parameter."""

    name: str
    location: Literal["path", "query", "header"]
    required: bool


@dataclass(frozen=True, slots=True)
class OperationSpec:
    """Describes an SDK operation generated from Fanvue API docs."""

    operation_id: str
    group: str
    method: str
    path: str
    summary: str
    description: str
    docs_url: str
    parameters: tuple[ParameterSpec, ...]
    request_body_required: bool
    request_body_content_types: tuple[str, ...]


Headers: TypeAlias = dict[str, str]
QueryParams: TypeAlias = dict[str, Any]
PathParams: TypeAlias = dict[str, Any]
