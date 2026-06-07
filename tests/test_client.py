"""Tests for transport, auth, header injection, and error mapping behaviour."""

from __future__ import annotations

import json
from typing import Any

import httpx
import pytest

from fanvue_sdk import FanvueAsyncClient
from fanvue_sdk.exceptions import (
    BadRequestError,
    FanvueAPIError,
    ForbiddenError,
    GoneError,
    NotFoundError,
    RateLimitError,
    UnauthorizedError,
)
from tests.conftest import ACCESS_TOKEN, API_VERSION, RequestRecorder, build_client

USER_PAYLOAD: dict[str, Any] = {
    "uuid": "32cbfcb4-676a-4e8c-8ea7-bf0de6173154",
    "email": "hello@example.com",
    "handle": "fanvue-user",
    "bio": "Creator bio",
    "displayName": "Fanvue User",
    "isCreator": True,
    "createdAt": "2026-02-22T00:00:00.000Z",
    "updatedAt": None,
    "avatarUrl": None,
    "bannerUrl": None,
}


def _user_response(request: httpx.Request) -> httpx.Response:
    return httpx.Response(status_code=200, json=USER_PAYLOAD)


@pytest.mark.asyncio
async def test_required_headers_are_injected(recorder: RequestRecorder) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        recorder.capture(request)
        return _user_response(request)

    client = build_client(handler)
    try:
        await client.users.get_current_user()
    finally:
        await client.aclose()

    assert recorder.captured.url.path == "/users/me"
    assert recorder.captured.headers["Authorization"] == f"Bearer {ACCESS_TOKEN}"
    assert recorder.captured.headers["X-Fanvue-API-Version"] == API_VERSION


@pytest.mark.asyncio
async def test_authorization_header_mode(recorder: RequestRecorder) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        recorder.capture(request)
        return _user_response(request)

    client = build_client(
        handler, access_token=None, authorization_header="Bearer direct-header"
    )
    try:
        await client.users.get_current_user()
    finally:
        await client.aclose()

    assert recorder.captured.headers["Authorization"] == "Bearer direct-header"


@pytest.mark.asyncio
async def test_sync_auth_provider(recorder: RequestRecorder) -> None:
    def provider() -> str:
        return "Bearer sync-provider-token"

    def handler(request: httpx.Request) -> httpx.Response:
        recorder.capture(request)
        return _user_response(request)

    client = build_client(handler, access_token=None, auth_header_provider=provider)
    try:
        await client.users.get_current_user()
    finally:
        await client.aclose()

    assert recorder.captured.headers["Authorization"] == "Bearer sync-provider-token"


@pytest.mark.asyncio
async def test_async_auth_provider(recorder: RequestRecorder) -> None:
    async def provider() -> str:
        return "Bearer async-provider-token"

    def handler(request: httpx.Request) -> httpx.Response:
        recorder.capture(request)
        return _user_response(request)

    client = build_client(handler, access_token=None, auth_header_provider=provider)
    try:
        await client.users.get_current_user()
    finally:
        await client.aclose()

    assert recorder.captured.headers["Authorization"] == "Bearer async-provider-token"


def test_exactly_one_auth_mode_is_required() -> None:
    transport = httpx.MockTransport(lambda request: httpx.Response(204))
    with pytest.raises(ValueError, match="exactly one auth mode"):
        FanvueAsyncClient(api_version=API_VERSION, transport=transport)
    with pytest.raises(ValueError, match="exactly one auth mode"):
        FanvueAsyncClient(
            api_version=API_VERSION,
            access_token="a",
            authorization_header="b",
            transport=transport,
        )


def test_api_version_must_be_non_empty() -> None:
    transport = httpx.MockTransport(lambda request: httpx.Response(204))
    with pytest.raises(ValueError, match="api_version"):
        FanvueAsyncClient(api_version="  ", access_token="a", transport=transport)


@pytest.mark.asyncio
async def test_context_manager_closes_transport() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return _user_response(request)

    async with build_client(handler) as client:
        user = await client.users.get_current_user()
    assert user.email == "hello@example.com"


@pytest.mark.asyncio
async def test_path_query_and_json_body_serialization(recorder: RequestRecorder) -> None:
    payload: dict[str, Any] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        recorder.capture(request)
        payload["path"] = request.url.path
        payload["query"] = dict(request.url.params.multi_items())
        payload["body"] = json.loads(request.content.decode("utf-8"))
        return httpx.Response(status_code=204)

    client = build_client(handler)
    try:
        result = await client.creators.complete_creator_upload_session(
            "creator-uuid",
            "upload-uuid",
            body={"parts": [{"partNumber": 1, "etag": "etag-value"}]},
        )
    finally:
        await client.aclose()

    assert result is None
    assert payload["path"] == "/creators/creator-uuid/media/uploads/upload-uuid"
    assert payload["query"] == {}
    assert payload["body"] == {"parts": [{"partNumber": 1, "etag": "etag-value"}]}


@pytest.mark.asyncio
async def test_none_query_params_are_dropped(recorder: RequestRecorder) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        recorder.capture(request)
        return httpx.Response(
            status_code=200, json={"data": [], "pagination": {"page": 1, "size": 20, "hasMore": False}}
        )

    client = build_client(handler)
    try:
        await client.subscribers.list_subscribers(page=2)
    finally:
        await client.aclose()

    params = recorder.captured.url.params
    assert params["page"] == "2"
    assert "size" not in params  # None-valued params must not be serialized


@pytest.mark.parametrize(
    ("status_code", "exception_type"),
    [
        (400, BadRequestError),
        (401, UnauthorizedError),
        (403, ForbiddenError),
        (404, NotFoundError),
        (410, GoneError),
        (429, RateLimitError),
        (500, FanvueAPIError),
    ],
)
@pytest.mark.asyncio
async def test_http_errors_map_to_typed_exceptions(
    status_code: int, exception_type: type[FanvueAPIError]
) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            status_code=status_code,
            json={"message": "boom"},
            headers={"x-request-id": "req-123"},
        )

    client = build_client(handler)
    try:
        with pytest.raises(exception_type) as error_info:
            await client.users.get_current_user()
    finally:
        await client.aclose()

    error = error_info.value
    assert error.status_code == status_code
    assert error.message == "boom"
    assert error.request_id == "req-123"


@pytest.mark.asyncio
async def test_low_level_request_helper(recorder: RequestRecorder) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        recorder.capture(request)
        return httpx.Response(status_code=200, json={"ok": True})

    client = build_client(handler)
    try:
        result = await client.request("GET", "/custom/endpoint", query_params={"a": "b"})
    finally:
        await client.aclose()

    assert result == {"ok": True}
    assert recorder.captured.url.path == "/custom/endpoint"
    assert recorder.captured.url.params["a"] == "b"
