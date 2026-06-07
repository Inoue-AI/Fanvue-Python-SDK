"""Tests for the core operations the Inoue AI platform relies on.

These mirror the parity surface of the Go SDK: current-user identity, post
read + publish, subscriber listing, and chat message publishing. Every
response is asserted to deserialize into its typed Pydantic model.
"""

from __future__ import annotations

import json
from typing import Any

import httpx
import pytest

from fanvue_sdk.models import (
    CreatePostResponse,
    GetCurrentUserResponse,
    GetPostByUuidResponse,
    GetPostsResponse,
    ListSubscribersResponse,
    SendMessageResponse,
)
from tests.conftest import RequestRecorder, build_client


def _post_payload(uuid: str, *, text: str | None = "Hello world") -> dict[str, Any]:
    """A spec-complete post object (every required field present)."""
    return {
        "uuid": uuid,
        "text": text,
        "audience": "subscribers",
        "collections": [],
        "commentsCount": 0,
        "createdAt": "2026-04-01T00:00:00.000Z",
        "expiresAt": None,
        "isPinned": False,
        "likesCount": 0,
        "mediaPreviewUuid": None,
        "mediaUuids": [],
        "price": None,
        "publishAt": None,
        "publishedAt": "2026-04-01T00:00:00.000Z",
        "tips": {"count": 0, "totalGross": 0, "totalNet": 0},
    }


def _subscriber(uuid: str) -> dict[str, Any]:
    """A spec-complete subscriber list item."""
    return {
        "uuid": uuid,
        "handle": "fan",
        "displayName": "Fan One",
        "nickname": None,
        "isTopSpender": False,
        "avatarUrl": None,
        "registeredAt": "2026-02-01T00:00:00.000Z",
    }


@pytest.mark.asyncio
async def test_get_current_user_typed_and_nullable_fields() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            status_code=200,
            json={
                "uuid": "u-1",
                "email": "creator@example.com",
                "handle": "creator",
                "bio": "",
                "displayName": "Creator",
                "isCreator": True,
                "createdAt": "2026-01-01T00:00:00.000Z",
                "updatedAt": None,
                "avatarUrl": None,
                "bannerUrl": None,
            },
        )

    client = build_client(handler)
    try:
        user = await client.users.get_current_user()
    finally:
        await client.aclose()

    assert isinstance(user, GetCurrentUserResponse)
    assert user.uuid == "u-1"
    assert user.email == "creator@example.com"
    assert user.avatarUrl is None  # nullable per spec; must not raise
    assert user.updatedAt is None


@pytest.mark.asyncio
async def test_get_posts_returns_paginated_typed_page(recorder: RequestRecorder) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        recorder.capture(request)
        return httpx.Response(
            status_code=200,
            json={
                "data": [_post_payload("post-1")],
                "pagination": {"page": 1, "size": 20, "hasMore": False},
            },
        )

    client = build_client(handler)
    try:
        page = await client.posts.get_posts(page=1, size=20)
    finally:
        await client.aclose()

    assert isinstance(page, GetPostsResponse)
    assert page.data[0].uuid == "post-1"
    assert page.pagination.hasMore is False
    assert recorder.captured.url.path == "/posts"
    assert recorder.captured.url.params["page"] == "1"


@pytest.mark.asyncio
async def test_get_post_by_uuid(recorder: RequestRecorder) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        recorder.capture(request)
        payload = _post_payload("post-42", text="Pinned")
        payload["isPinned"] = True
        payload["likesCount"] = 7
        payload["commentsCount"] = 2
        return httpx.Response(status_code=200, json=payload)

    client = build_client(handler)
    try:
        post = await client.posts.get_post_by_uuid("post-42")
    finally:
        await client.aclose()

    assert isinstance(post, GetPostByUuidResponse)
    assert post.uuid == "post-42"
    assert post.isPinned is True
    assert recorder.captured.url.path == "/posts/post-42"


@pytest.mark.asyncio
async def test_create_post_sends_body_and_returns_typed(recorder: RequestRecorder) -> None:
    captured_body: dict[str, Any] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        recorder.capture(request)
        captured_body.update(json.loads(request.content.decode("utf-8")))
        return httpx.Response(status_code=201, json=_post_payload("post-new", text="Launch day!"))

    client = build_client(handler)
    try:
        created = await client.posts.create_post(
            body={"text": "Launch day!", "audience": "subscribers", "mediaUuids": ["m-1"]}
        )
    finally:
        await client.aclose()

    assert isinstance(created, CreatePostResponse)
    assert created.uuid == "post-new"
    assert recorder.captured.method == "POST"
    assert recorder.captured.url.path == "/posts"
    assert captured_body["text"] == "Launch day!"
    assert captured_body["mediaUuids"] == ["m-1"]


@pytest.mark.asyncio
async def test_list_subscribers(recorder: RequestRecorder) -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        recorder.capture(request)
        return httpx.Response(
            status_code=200,
            json={
                "data": [_subscriber("fan-1")],
                "pagination": {"page": 1, "size": 20, "hasMore": True},
            },
        )

    client = build_client(handler)
    try:
        subs = await client.subscribers.list_subscribers(page=1, size=20)
    finally:
        await client.aclose()

    assert isinstance(subs, ListSubscribersResponse)
    assert subs.data[0].uuid == "fan-1"
    assert subs.pagination.hasMore is True
    assert recorder.captured.url.path == "/subscribers"


@pytest.mark.asyncio
async def test_send_message_publishes_to_chat(recorder: RequestRecorder) -> None:
    captured_body: dict[str, Any] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        recorder.capture(request)
        captured_body.update(json.loads(request.content.decode("utf-8")))
        return httpx.Response(status_code=201, json={"messageUuid": "msg-1"})

    client = build_client(handler)
    try:
        sent = await client.chats.send_message(
            "fan-uuid", body={"text": "Thanks for subscribing!"}
        )
    finally:
        await client.aclose()

    assert isinstance(sent, SendMessageResponse)
    assert sent.messageUuid == "msg-1"
    assert recorder.captured.method == "POST"
    assert recorder.captured.url.path == "/chats/fan-uuid/message"
    assert captured_body["text"] == "Thanks for subscribing!"


@pytest.mark.asyncio
async def test_missing_required_body_raises_before_request() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        raise AssertionError("request must not be sent when a required body is missing")

    client = build_client(handler)
    try:
        with pytest.raises(ValueError, match="requires a request body"):
            await client.posts.create_post(body=None)
    finally:
        await client.aclose()
