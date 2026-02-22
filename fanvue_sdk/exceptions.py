from __future__ import annotations

from typing import Any


class FanvueAPIError(Exception):
    """Base exception for all API errors raised by the SDK."""

    def __init__(
        self,
        *,
        status_code: int,
        message: str,
        response_data: Any = None,
        request_id: str | None = None,
    ) -> None:
        self.status_code = status_code
        self.message = message
        self.response_data = response_data
        self.request_id = request_id
        super().__init__(self.__str__())

    def __str__(self) -> str:
        details = f"{self.status_code}: {self.message}"
        if self.request_id:
            return f"{details} (request_id={self.request_id})"
        return details


class BadRequestError(FanvueAPIError):
    """400 error from Fanvue API."""


class UnauthorizedError(FanvueAPIError):
    """401 error from Fanvue API."""


class ForbiddenError(FanvueAPIError):
    """403 error from Fanvue API."""


class NotFoundError(FanvueAPIError):
    """404 error from Fanvue API."""


class GoneError(FanvueAPIError):
    """410 error from Fanvue API."""


class RateLimitError(FanvueAPIError):
    """429 error from Fanvue API."""
