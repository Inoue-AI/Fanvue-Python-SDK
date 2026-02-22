from __future__ import annotations

import json
from typing import Any

import httpx
import pytest

from fanvue_sdk import FanvueAsyncClient
from fanvue_sdk.exceptions import ForbiddenError
from fanvue_sdk.models import GetCurrentUserResponse, ListMediaResponse


@pytest.mark.asyncio
async def test_get_current_user_sends_required_headers() -> None:
    captured_request: httpx.Request | None = None

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal captured_request
        captured_request = request
        return httpx.Response(
            status_code=200,
            json={
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
            },
        )

    transport = httpx.MockTransport(handler)
    client = FanvueAsyncClient(
        api_version="2025-06-26",
        access_token="token-value",
        transport=transport,
    )

    try:
        response = await client.users.get_current_user()
    finally:
        await client.aclose()

    assert isinstance(response, GetCurrentUserResponse)
    assert response.uuid == "32cbfcb4-676a-4e8c-8ea7-bf0de6173154"
    assert response.email == "hello@example.com"
    assert captured_request is not None
    assert captured_request.url.path == "/users/me"
    assert captured_request.headers["Authorization"] == "Bearer token-value"
    assert captured_request.headers["X-Fanvue-API-Version"] == "2025-06-26"


@pytest.mark.asyncio
async def test_path_query_and_json_body_are_serialized() -> None:
    payload: dict[str, Any] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        payload["path"] = request.url.path
        payload["query"] = dict(request.url.params.multi_items())
        payload["body"] = json.loads(request.content.decode("utf-8"))
        return httpx.Response(status_code=204)

    transport = httpx.MockTransport(handler)
    client = FanvueAsyncClient(
        api_version="2025-06-26",
        authorization_header="Bearer direct-header",
        transport=transport,
    )

    try:
        response = await client.agencies.complete_creator_upload_session(
            "creator-uuid",
            "upload-uuid",
            body={"parts": [{"partNumber": 1, "etag": "etag-value"}]},
        )
    finally:
        await client.aclose()

    assert response is None
    assert payload == {
        "path": "/creators/creator-uuid/media/uploads/upload-uuid",
        "query": {},
        "body": {"parts": [{"partNumber": 1, "etag": "etag-value"}]},
    }


@pytest.mark.asyncio
async def test_query_parameters_are_sent_for_get_operations() -> None:
    captured_request: httpx.Request | None = None

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal captured_request
        captured_request = request
        return httpx.Response(
            status_code=200,
            json={
                "data": [
                    {
                        "uuid": "24ba4adc-4319-4af4-af02-6f4b1190f56e",
                        "messageUuid": "a76d92ff-e84e-4a5d-aafe-bc6bd2e30cdf",
                        "mediaType": "image",
                        "created_at": "2026-02-22T00:00:00.000Z",
                        "sentAt": "2026-02-22T00:00:00.000Z",
                        "ownerUuid": "1bd0f6ee-af58-486a-a855-39091aef5ffc",
                        "name": None,
                    }
                ],
                "nextCursor": None,
            },
        )

    transport = httpx.MockTransport(handler)
    client = FanvueAsyncClient(
        api_version="2025-06-26",
        access_token="token-value",
        transport=transport,
    )

    try:
        response = await client.chats.list_media(
            "0b3160a8-b173-4b58-bfcd-63f9f9d07cf6",
            cursor="next-page",
            limit=25,
        )
    finally:
        await client.aclose()

    assert isinstance(response, ListMediaResponse)
    assert response.data[0].mediaType == "image"
    assert captured_request is not None
    assert captured_request.url.path == "/chats/0b3160a8-b173-4b58-bfcd-63f9f9d07cf6/media"
    assert captured_request.url.params["cursor"] == "next-page"
    assert captured_request.url.params["limit"] == "25"


@pytest.mark.asyncio
async def test_http_errors_are_mapped_to_sdk_exceptions() -> None:
    def handler(_: httpx.Request) -> httpx.Response:
        return httpx.Response(status_code=403, json={"message": "Forbidden"})

    transport = httpx.MockTransport(handler)
    client = FanvueAsyncClient(
        api_version="2025-06-26",
        access_token="token-value",
        transport=transport,
    )

    try:
        with pytest.raises(ForbiddenError) as error:
            await client.users.get_current_user()
    finally:
        await client.aclose()

    assert error.value.status_code == 403
    assert error.value.message == "Forbidden"


@pytest.mark.asyncio
async def test_async_auth_provider_is_supported() -> None:
    captured_request: httpx.Request | None = None

    async def provider() -> str:
        return "Bearer provider-token"

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal captured_request
        captured_request = request
        return httpx.Response(
            status_code=200,
            json={
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
            },
        )

    transport = httpx.MockTransport(handler)
    client = FanvueAsyncClient(
        api_version="2025-06-26",
        auth_header_provider=provider,
        transport=transport,
    )

    try:
        response = await client.users.get_current_user()
    finally:
        await client.aclose()

    assert isinstance(response, GetCurrentUserResponse)
    assert captured_request is not None
    assert captured_request.headers["Authorization"] == "Bearer provider-token"
