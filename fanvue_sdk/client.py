from __future__ import annotations

from collections.abc import Awaitable, Callable, Mapping
from inspect import isawaitable
from typing import Any, cast
from urllib.parse import quote

import httpx

from fanvue_sdk._operations import OPERATIONS
from fanvue_sdk.exceptions import (
    BadRequestError,
    FanvueAPIError,
    ForbiddenError,
    GoneError,
    NotFoundError,
    RateLimitError,
    UnauthorizedError,
)
from fanvue_sdk.models import parse_operation_response
from fanvue_sdk.resources import RESOURCE_FACTORIES
from fanvue_sdk.types import OperationSpec, ResponseData

AuthHeaderProvider = Callable[[], str | Awaitable[str]]

_ERROR_MAP: dict[int, type[FanvueAPIError]] = {
    400: BadRequestError,
    401: UnauthorizedError,
    403: ForbiddenError,
    404: NotFoundError,
    410: GoneError,
    429: RateLimitError,
}


class FanvueAsyncClient:
    """Async Fanvue API client with generated resource methods."""

    def __init__(
        self,
        *,
        api_version: str,
        access_token: str | None = None,
        authorization_header: str | None = None,
        auth_header_provider: AuthHeaderProvider | None = None,
        base_url: str = "https://api.fanvue.com",
        timeout: float = 30.0,
        default_headers: Mapping[str, str] | None = None,
        transport: httpx.AsyncBaseTransport | None = None,
    ) -> None:
        if not api_version.strip():
            raise ValueError("api_version must be provided")

        auth_mode_count = sum(
            [
                access_token is not None,
                authorization_header is not None,
                auth_header_provider is not None,
            ]
        )
        if auth_mode_count != 1:
            raise ValueError(
                "Provide exactly one auth mode: access_token, authorization_header, or auth_header_provider"
            )

        self._api_version = api_version
        self._access_token = access_token
        self._authorization_header = authorization_header
        self._auth_header_provider = auth_header_provider
        self._default_headers = dict(default_headers or {})
        self._closed = False

        normalized_base_url = base_url.rstrip("/")
        self._http = httpx.AsyncClient(
            base_url=normalized_base_url,
            timeout=timeout,
            transport=transport,
        )

        for resource_name, resource_factory in RESOURCE_FACTORIES.items():
            setattr(self, resource_name, resource_factory(self))

    async def __aenter__(self) -> FanvueAsyncClient:
        return self

    async def __aexit__(self, exc_type: Any, exc: Any, tb: Any) -> None:
        await self.aclose()

    async def aclose(self) -> None:
        if self._closed:
            return
        await self._http.aclose()
        self._closed = True

    @property
    def api_version(self) -> str:
        return self._api_version

    async def request(
        self,
        method: str,
        path: str,
        *,
        query_params: Mapping[str, Any] | None = None,
        headers: Mapping[str, str] | None = None,
        json_body: Any = None,
        data_body: Any = None,
    ) -> ResponseData:
        """Low-level request helper for custom endpoints not covered by generated methods."""
        return await self._request(
            method=method,
            path=path,
            query_params=query_params,
            headers=headers,
            json_body=json_body,
            data_body=data_body,
        )

    async def _call_operation(
        self,
        operation_id: str,
        *,
        path_params: Mapping[str, Any] | None = None,
        query_params: Mapping[str, Any] | None = None,
        header_params: Mapping[str, Any] | None = None,
        body: Mapping[str, Any] | None = None,
    ) -> Any:
        operation = OPERATIONS.get(operation_id)
        if operation is None:
            raise ValueError(f"Unknown operation_id: {operation_id}")

        normalized_path = self._validate_and_render_path(operation, path_params or {})
        normalized_query = self._validate_params(operation, query_params or {}, location="query")
        normalized_headers = self._validate_params(operation, header_params or {}, location="header")

        json_body: Mapping[str, Any] | None = None
        data_body: Mapping[str, Any] | None = None

        if operation.request_body_required and body is None:
            raise ValueError(f"Operation '{operation.operation_id}' requires a request body")

        if body is not None and not operation.request_body_content_types:
            raise ValueError(f"Operation '{operation.operation_id}' does not support a request body")

        if body is not None:
            if "application/json" in operation.request_body_content_types:
                json_body = body
            else:
                data_body = body

        raw_response = await self._request(
            method=operation.method,
            path=normalized_path,
            query_params=normalized_query,
            headers=normalized_headers,
            json_body=json_body,
            data_body=data_body,
        )
        return parse_operation_response(operation.operation_id, raw_response)

    def _validate_and_render_path(self, operation: OperationSpec, path_params: Mapping[str, Any]) -> str:
        expected_path_params = [
            parameter for parameter in operation.parameters if parameter.location == "path"
        ]
        expected_names = {parameter.name for parameter in expected_path_params}

        unexpected = set(path_params) - expected_names
        if unexpected:
            invalid = ", ".join(sorted(unexpected))
            raise ValueError(f"Unexpected path parameters for '{operation.operation_id}': {invalid}")

        missing_required = [
            parameter.name
            for parameter in expected_path_params
            if parameter.required and (parameter.name not in path_params or path_params[parameter.name] is None)
        ]
        if missing_required:
            invalid = ", ".join(sorted(missing_required))
            raise ValueError(
                f"Missing required path parameters for '{operation.operation_id}': {invalid}"
            )

        rendered_path = operation.path
        for parameter in expected_path_params:
            if parameter.name not in path_params:
                continue
            value = path_params[parameter.name]
            if value is None:
                continue
            rendered_path = rendered_path.replace(
                f"{{{parameter.name}}}", quote(str(value), safe=""),
            )

        return rendered_path

    def _validate_params(
        self,
        operation: OperationSpec,
        provided_params: Mapping[str, Any],
        *,
        location: str,
    ) -> dict[str, Any]:
        expected = [parameter for parameter in operation.parameters if parameter.location == location]
        expected_names = {parameter.name for parameter in expected}

        unexpected = set(provided_params) - expected_names
        if unexpected:
            invalid = ", ".join(sorted(unexpected))
            raise ValueError(
                f"Unexpected {location} parameters for '{operation.operation_id}': {invalid}"
            )

        missing_required = [
            parameter.name
            for parameter in expected
            if parameter.required
            and (parameter.name not in provided_params or provided_params[parameter.name] is None)
        ]
        if missing_required:
            invalid = ", ".join(sorted(missing_required))
            raise ValueError(
                f"Missing required {location} parameters for '{operation.operation_id}': {invalid}"
            )

        return {
            key: value
            for key, value in provided_params.items()
            if value is not None and key in expected_names
        }

    async def _request(
        self,
        *,
        method: str,
        path: str,
        query_params: Mapping[str, Any] | None = None,
        headers: Mapping[str, Any] | None = None,
        json_body: Any = None,
        data_body: Any = None,
    ) -> ResponseData:
        request_headers = await self._build_headers(headers)

        response = await self._http.request(
            method=method,
            url=path,
            params=query_params,
            headers=request_headers,
            json=json_body,
            data=data_body,
        )

        if response.status_code >= 400:
            self._raise_api_error(response)

        if not response.content:
            return None

        content_type = response.headers.get("content-type", "")
        if "application/json" in content_type:
            return cast(ResponseData, response.json())

        return response.content

    async def _build_headers(self, headers: Mapping[str, Any] | None) -> dict[str, str]:
        authorization = await self._resolve_authorization_header()
        request_headers: dict[str, str] = {
            "Authorization": authorization,
            "X-Fanvue-API-Version": self._api_version,
        }
        request_headers.update(self._default_headers)

        if headers:
            request_headers.update({key: str(value) for key, value in headers.items() if value is not None})

        return request_headers

    async def _resolve_authorization_header(self) -> str:
        if self._access_token is not None:
            return f"Bearer {self._access_token}"

        if self._authorization_header is not None:
            return self._authorization_header

        if self._auth_header_provider is None:
            raise RuntimeError("No auth header provider configured")

        header_value = self._auth_header_provider()
        if isawaitable(header_value):
            header_value = await header_value

        if not isinstance(header_value, str) or not header_value.strip():
            raise ValueError("auth_header_provider must return a non-empty string")

        return header_value

    def _raise_api_error(self, response: httpx.Response) -> None:
        response_data: Any
        try:
            response_data = response.json()
        except ValueError:
            response_data = response.text

        message = _extract_error_message(response_data, fallback=response.reason_phrase)
        error_type = _ERROR_MAP.get(response.status_code, FanvueAPIError)

        raise error_type(
            status_code=response.status_code,
            message=message,
            response_data=response_data,
            request_id=response.headers.get("x-request-id"),
        )


def _extract_error_message(response_data: Any, *, fallback: str) -> str:
    if isinstance(response_data, dict):
        for key in ("message", "error", "detail", "title"):
            value = response_data.get(key)
            if isinstance(value, str) and value.strip():
                return value

    if isinstance(response_data, str) and response_data.strip():
        return response_data

    if fallback:
        return fallback

    return "Fanvue API request failed"
