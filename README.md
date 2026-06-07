# Fanvue SDK

Typed clients for the [Fanvue API](https://api.fanvue.com) in two languages
that live side by side in this repository:

- **Python** (`fanvue_sdk/`) — fully async, exhaustive coverage of every
  documented endpoint, generated from a vendored OpenAPI snapshot.
- **Go** (`go/`) — idiomatic, context-aware, covering the core operations the
  Inoue AI platform consumes, at behavioural parity with the Python client.

[![PyPI](https://img.shields.io/pypi/v/fanvue-python-sdk)](https://pypi.org/project/fanvue-python-sdk/)
[![Python](https://img.shields.io/pypi/pyversions/fanvue-python-sdk)](https://pypi.org/project/fanvue-python-sdk/)

| | Python | Go |
|---|---|---|
| Location | `fanvue_sdk/` | `go/` |
| Coverage | All 143 documented operations | Core ops (identity, posts, fans, chat) |
| Source of truth | `fanvue_sdk/_spec/openapi.json` (vendored) | Hand-maintained, mirrors Python |
| Runtime | `httpx.AsyncClient` | one `*http.Client`, ctx-first |
| Tests | `pytest` + `httpx.MockTransport` | `go test` + `httptest` |

Both clients require an externally-obtained OAuth access token; neither performs
the OAuth handshake or token refresh (Fanvue does not expose a refresh endpoint
to API consumers).

---

## Python SDK

### Requirements

- Python `>=3.11`

### Installation

```bash
pip install fanvue-python-sdk
# development tools:
pip install -e .[dev]
```

### Quickstart

```python
import asyncio

from fanvue_sdk import FanvueAsyncClient


async def main() -> None:
    async with FanvueAsyncClient(
        api_version="2025-06-26",
        access_token="<oauth-access-token>",
    ) as client:
        me = await client.users.get_current_user()
        print(me.uuid, me.email)

        page = await client.posts.get_posts(page=1, size=20)
        print(len(page.data), "posts")

        created = await client.posts.create_post(
            body={"audience": "subscribers", "text": "Hello fans!"}
        )
        print("created", created.uuid)


if __name__ == "__main__":
    asyncio.run(main())
```

### Authentication

OAuth token management is external. Provide exactly one of:

1. `access_token` — the SDK sends `Authorization: Bearer <token>`.
2. `authorization_header` — the full header value, sent verbatim.
3. `auth_header_provider` — a sync or async callable returning the header value
   (use this for token rotation).

The SDK always sends `X-Fanvue-API-Version` using the `api_version` you pass.

### Resource groups

`client.<group>.<operation>(...)`. Groups are derived from the API path and
include: `users`, `posts`, `chats`, `subscribers`, `followers`, `creators`,
`agencies`, `apps`, `collections`, `insights`, `media`, `notifications`,
`tracking_links`, `vault`. Each method returns a typed Pydantic model (or
`None` for 204 endpoints). See [`docs-endpoints.md`](./docs-endpoints.md) for
the full operation index.

### Regenerating from the OpenAPI spec

The Python SDK is generated from a **vendored snapshot** of Fanvue's
consolidated OpenAPI document at
[`fanvue_sdk/_spec/openapi.json`](./fanvue_sdk/_spec/openapi.json). Generation
is fully reproducible and works offline:

```bash
python scripts/generate_sdk.py            # generate from the vendored snapshot
python scripts/generate_sdk.py --refresh  # re-download the spec, then generate
```

Generated files (all marked auto-generated): `fanvue_sdk/_operations.py`,
`fanvue_sdk/models.py`, `fanvue_sdk/resources/*.py` (except `base.py`),
`fanvue_sdk/resources/__init__.py`, and `docs-endpoints.md`. The vendored spec
is the source of truth; commit it together with the generated output so CI can
verify they stay in sync.

### Development commands

```bash
ruff check .
mypy
pytest
```

### Docker

```bash
docker compose up --build      # build + run the Python test suite
```

---

## Go SDK

The Go client lives in [`go/`](./go) and exposes the core operations the Inoue
AI backend uses. See [`go/README.md`](./go/README.md) for full details.

```go
client, err := fanvue.New(fanvue.ClientOptions{
    APIVersion:  "2025-06-26",
    AccessToken: "<oauth-access-token>",
})
if err != nil {
    log.Fatal(err)
}
defer client.Close()

user, err := client.GetCurrentUser(ctx)
```

```bash
cd go
go build ./...
go vet ./...
go test ./...
```

---

## Notes

- **Live verification is credential-gated.** The test suites mock all HTTP
  traffic (no real Fanvue calls, no credentials required). End-to-end
  verification against the live API requires a valid OAuth token and has not
  been performed here.
- **The vendored spec is a point-in-time snapshot.** Re-run
  `scripts/generate_sdk.py --refresh` to pick up upstream API changes.
