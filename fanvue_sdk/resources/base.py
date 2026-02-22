from __future__ import annotations

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from fanvue_sdk.client import FanvueAsyncClient


class BaseResource:
    """Base class shared by all generated resources."""

    def __init__(self, client: FanvueAsyncClient) -> None:
        self._client = client
