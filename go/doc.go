// Package fanvue provides a typed, context-aware Go client for the Fanvue API
// (https://api.fanvue.com).
//
// It mirrors the core surface of the Python SDK that lives alongside it in the
// same repository, so backend services can swap implementations without
// behavioural drift. The Go client intentionally covers the operations the
// Inoue AI platform consumes — current-user identity, post read + publish,
// subscriber/follower listing, and chat message publishing — rather than the
// full 143-operation Fanvue surface. Additional endpoints can be added
// incrementally without changing the client core.
//
// Operating principles (matching the Inoue AI Go backend memory-safety bar):
//
//   - Every public method takes context.Context as its first parameter and
//     propagates cancellation to the underlying HTTP call.
//   - Each *Client owns one *http.Client with an explicit Timeout, bounded
//     idle connections, and an IdleConnTimeout. http.DefaultClient is never
//     used.
//   - defer client.Close() releases idle connections.
//   - Response bodies are always closed; errors are wrapped with context.
//
// Authentication is OAuth-token based and handled externally: construct the
// client with a bearer access token (or a full Authorization header, or a
// dynamic header provider for token rotation). Fanvue does not expose a
// token-refresh endpoint to API consumers, so the SDK never performs the
// refresh itself — the caller supplies a valid token.
//
//	client := fanvue.New(fanvue.ClientOptions{
//	    APIVersion:  "2025-06-26",
//	    AccessToken: "OAUTH_ACCESS_TOKEN",
//	})
//	defer client.Close()
//
//	user, err := client.GetCurrentUser(ctx)
//	if err != nil {
//	    return err
//	}
package fanvue
