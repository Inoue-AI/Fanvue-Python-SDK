# Fanvue Go SDK

Typed, context-aware Go client for the [Fanvue API](https://api.fanvue.com).

This SDK lives alongside the Python SDK in the same repository and exposes a
focused subset of the platform: the operations the Inoue AI backend consumes
(current-user identity, post read + publish, subscriber/follower listing, and
chat message publishing). It mirrors the Python client's behaviour so backend
services can swap implementations without drift.

## Install

```bash
go get github.com/Inoue-AI/Fanvue-Python-SDK/go@latest
```

## Quickstart

```go
package main

import (
	"context"
	"log"
	"time"

	fanvue "github.com/Inoue-AI/Fanvue-Python-SDK/go"
)

func main() {
	client, err := fanvue.New(fanvue.ClientOptions{
		APIVersion:  "2025-06-26",
		AccessToken: "OAUTH_ACCESS_TOKEN",
		Timeout:     30 * time.Second,
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		log.Fatalf("get current user: %v", err)
	}
	log.Printf("authenticated as %s (%s)", user.Handle, user.UUID)
}
```

## Authentication

Supply exactly one auth mode when constructing the client (mirroring the Python
SDK). `APIVersion` is always required and sent as `X-Fanvue-API-Version`.

| Mode | Field | Behaviour |
|---|---|---|
| Bearer token | `AccessToken` | Sends `Authorization: Bearer <token>` |
| Full header | `AuthorizationHeader` | Sends the value verbatim |
| Dynamic | `AuthHeaderProvider` | Called per request (token rotation) |

Fanvue does not expose a token-refresh endpoint to API consumers, so the SDK
never refreshes tokens itself — the caller supplies a valid OAuth token (use
`AuthHeaderProvider` to rotate it transparently).

## Methods (core parity surface)

| Go method | Fanvue endpoint | Scope |
|---|---|---|
| `GetCurrentUser(ctx)` | `GET /users/me` | `read:self` |
| `ListPosts(ctx, params)` | `GET /posts` | `read:post` |
| `GetPost(ctx, uuid)` | `GET /posts/{uuid}` | `read:post` |
| `CreatePost(ctx, params)` | `POST /posts` | `write:post` |
| `ListSubscribers(ctx, params)` | `GET /subscribers` | `read:fan` |
| `ListFollowers(ctx, params)` | `GET /followers` | `read:fan` |
| `ListMessages(ctx, uuid, params)` | `GET /chats/{userUuid}/messages` | `read:chat` |
| `SendMessage(ctx, uuid, params)` | `POST /chats/{userUuid}/message` | `write:chat` |

The full Fanvue surface is 143 operations; the remaining endpoints (agency /
creator-scoped operations, media uploads, vault, insights, tracking links,
collections, notifications) are covered exhaustively by the Python SDK and can
be added to the Go client incrementally without changing its core.

## Operating principles

The Go client is built to the same memory-safety bar as the Inoue AI Go
backend:

- Every method takes `context.Context` first; cancellation propagates to the
  underlying HTTP call.
- Each `*Client` owns one `*http.Client` with an explicit `Timeout`,
  `MaxIdleConnsPerHost`, and `IdleConnTimeout`. `http.DefaultClient` is never
  used.
- `defer client.Close()` releases idle connections.
- Response bodies are always closed; errors are wrapped with context.

## Errors

Non-2xx responses surface as `*fanvue.Error` with `StatusCode`, `Message`,
field-level `Errors`, and `RequestID`:

```go
post, err := client.CreatePost(ctx, params)
if err != nil {
	if apiErr, ok := fanvue.AsError(err); ok {
		switch {
		case apiErr.IsUnauthorized():
			// supply a fresh token
		case apiErr.IsRateLimited():
			// back off
		case apiErr.IsServerError():
			// retry
		}
	}
	return err
}
```

## Development

```bash
go build ./...
go vet ./...
go test ./...          # add -race in CI (requires cgo)
gofmt -l .             # must print nothing
```

## Repository layout

The Go SDK lives in the `go/` subdirectory. The Python SDK remains under
`fanvue_sdk/`. See the top-level [README](../README.md) for the cross-language
overview.
