# Fanvue Python SDK

Async Python SDK for the [Fanvue API](https://api.fanvue.com/docs), generated from Fanvue's official API reference (`.mdx`) endpoints.

[![PyPI](https://img.shields.io/pypi/v/fanvue-python-sdk)](https://pypi.org/project/fanvue-python-sdk/)
[![Python](https://img.shields.io/pypi/pyversions/fanvue-python-sdk)](https://pypi.org/project/fanvue-python-sdk/)

- Fully async (`httpx.AsyncClient`)
- Generated resource methods for all documented API reference endpoints
- Generated Pydantic response models for endpoint payloads
- No SDK-side fallback values injected into API response payloads
- OAuth handled externally (pass token/header/provider)

## Requirements

- Python `>=3.11`

## Installation

```bash
pip install fanvue-python-sdk
```

For development tools:

```bash
pip install -e .[dev]
```

## Quickstart

```python
import asyncio

from fanvue_sdk import FanvueAsyncClient


async def main() -> None:
    async with FanvueAsyncClient(
        api_version="2025-06-26",
        access_token="<oauth-access-token>",
    ) as client:
        me = await client.users.get_current_user()
        print(type(me).__name__)
        print(me.email)
        print(me.model_dump())


if __name__ == "__main__":
    asyncio.run(main())
```

## Authentication

OAuth flow/token management is external. Provide exactly one of:

1. `access_token` (SDK sends `Authorization: Bearer <token>`)
2. `authorization_header` (full header value)
3. `auth_header_provider` (sync or async callable returning header value)

The SDK always sends `X-Fanvue-API-Version` using the `api_version` you pass.

## Resource Groups

The generated client exposes these resources:

- `users`
- `chats`
- `chat_messages`
- `chat_templates`
- `chat_smart_lists`
- `chat_custom_lists`
- `chat_custom_list_members`
- `posts`
- `creators`
- `insights`
- `media`
- `tracking_links`
- `vault`
- `agencies`

A generated operation index is available in `docs-endpoints.md`.

## Methods and Typed Responses

- Operation methods live in `fanvue_sdk/resources/*.py` (exposed as `client.<resource>.<operation>(...)`).
- Response models are generated in `fanvue_sdk/models.py`.
- Each method returns a typed Pydantic response model (or `None` for 204 endpoints).

Example:

```python
me = await client.users.get_current_user()
print(me.displayName)
payload = me.model_dump()  # reuse this for external APIs
```

## Regenerating From Fanvue Docs

The SDK code is generated from Fanvue's official docs listing:

- index: `https://api.fanvue.com/docs/llms.txt`
- endpoint pages: `https://api.fanvue.com/docs/api-reference/reference/.../*.mdx`

Regenerate operations and resources:

```bash
python3 scripts/generate_sdk.py
```

Generated files:

- `fanvue_sdk/_operations.py`
- `fanvue_sdk/models.py`
- `fanvue_sdk/resources/*.py` (except `base.py`)
- `fanvue_sdk/resources/__init__.py`
- `docs-endpoints.md`

## Docker

Build and run tests:

```bash
docker compose up --build
```

Or run with plain Docker:

```bash
docker build -t fanvue-python-sdk .
docker run --rm fanvue-python-sdk
```

## Development Commands

```bash
ruff check .
mypy
pytest
```
