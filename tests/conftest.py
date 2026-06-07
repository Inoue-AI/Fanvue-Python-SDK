"""Shared pytest fixtures and helpers for the Fanvue SDK test suite.

Every test runs against an in-process :class:`httpx.MockTransport`; no real
network calls are ever made and no credentials are required.
"""

from __future__ import annotations

from collections.abc import Callable, Iterator

import httpx
import pytest

from fanvue_sdk import FanvueAsyncClient

API_VERSION = "2025-06-26"
ACCESS_TOKEN = "test-access-token"

Handler = Callable[[httpx.Request], httpx.Response]


def build_client(handler: Handler, **overrides: object) -> FanvueAsyncClient:
    """Construct a client whose HTTP calls are served by ``handler``.

    Keyword overrides are forwarded to :class:`FanvueAsyncClient`, letting a
    test swap the auth mode (``access_token`` is supplied by default).
    """
    kwargs: dict[str, object] = {
        "api_version": API_VERSION,
        "access_token": ACCESS_TOKEN,
        "transport": httpx.MockTransport(handler),
    }
    kwargs.update(overrides)
    return FanvueAsyncClient(**kwargs)  # type: ignore[arg-type]


class RequestRecorder:
    """Captures the most recent request a handler received, for assertions."""

    def __init__(self) -> None:
        self.request: httpx.Request | None = None

    def capture(self, request: httpx.Request) -> None:
        self.request = request

    @property
    def captured(self) -> httpx.Request:
        if self.request is None:
            raise AssertionError("no request was captured")
        return self.request


@pytest.fixture
def recorder() -> Iterator[RequestRecorder]:
    """Provide a fresh :class:`RequestRecorder` per test."""
    yield RequestRecorder()
