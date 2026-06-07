package fanvue

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// VaultFolder is a single vault folder owned by the authenticated user. It is
// the shape returned by ListVaultFolders (per item), CreateVaultFolder, and
// GetVaultFolder, all of which share the same payload. CreatedAt is nil when the
// API reports a JSON null.
type VaultFolder struct {
	CreatedAt  *string `json:"createdAt"`
	MediaCount float64 `json:"mediaCount"`
	Name       string  `json:"name"`
}

// VaultFoldersPage is one page of the authenticated user's vault folders.
type VaultFoldersPage struct {
	Data       []VaultFolder `json:"data"`
	Pagination Pagination    `json:"pagination"`
}

// ListVaultFolders returns the authenticated user's vault folders.
//
// GET /vault/folders — scope: read:vault.
func (c *Client) ListVaultFolders(ctx context.Context) (*VaultFoldersPage, error) {
	out := &VaultFoldersPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/vault/folders", nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateVaultFolder creates a vault folder. Body is the opaque request payload
// (mirroring the Python SDK's Mapping[str, Any] signature) and must be supplied
// — typically {"name": "..."}.
//
// POST /vault/folders — scope: write:vault.
func (c *Client) CreateVaultFolder(ctx context.Context, body RawBody) (*VaultFolder, error) {
	if len(body) == 0 {
		return nil, errors.New("fanvue: CreateVaultFolder requires a non-empty body")
	}
	out := &VaultFolder{}
	if err := c.doJSON(ctx, http.MethodPost, "/vault/folders", nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetVaultFolder fetches a single vault folder by its name.
//
// GET /vault/folders/{folderName} — scope: read:vault.
func (c *Client) GetVaultFolder(ctx context.Context, folderName string) (*VaultFolder, error) {
	if folderName == "" {
		return nil, errors.New("fanvue: GetVaultFolder requires a non-empty folder name")
	}
	path := "/vault/folders/" + url.PathEscape(folderName)
	out := &VaultFolder{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// RenameVaultFolder renames a vault folder. Body is the opaque request payload
// (mirroring the Python SDK's Mapping[str, Any] signature) and must be supplied
// — typically {"name": "..."}. The endpoint returns no body.
//
// PATCH /vault/folders/{folderName} — scope: write:vault.
func (c *Client) RenameVaultFolder(ctx context.Context, folderName string, body RawBody) error {
	if folderName == "" {
		return errors.New("fanvue: RenameVaultFolder requires a non-empty folder name")
	}
	if len(body) == 0 {
		return errors.New("fanvue: RenameVaultFolder requires a non-empty body")
	}
	path := "/vault/folders/" + url.PathEscape(folderName)
	return c.doJSON(ctx, http.MethodPatch, path, nil, body, nil)
}

// DeleteVaultFolder deletes a vault folder by name. The endpoint returns no
// body.
//
// DELETE /vault/folders/{folderName} — scope: write:vault.
func (c *Client) DeleteVaultFolder(ctx context.Context, folderName string) error {
	if folderName == "" {
		return errors.New("fanvue: DeleteVaultFolder requires a non-empty folder name")
	}
	path := "/vault/folders/" + url.PathEscape(folderName)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// VaultFolderMediaItem is one media item inside a vault folder, as returned by
// ListVaultFolderMedia. It is a lightweight projection (uuid, type, name); use
// the media endpoints for full details and variant URLs. CreatedAt and Name are
// nil when the API reports a JSON null.
type VaultFolderMediaItem struct {
	CreatedAt *string `json:"createdAt"`
	MediaType string  `json:"mediaType"`
	Name      *string `json:"name"`
	UUID      string  `json:"uuid"`
}

// VaultFolderMediaPage is one page of media inside a vault folder.
type VaultFolderMediaPage struct {
	Data       []VaultFolderMediaItem `json:"data"`
	Pagination Pagination             `json:"pagination"`
}

// ListVaultFolderMediaParams configures a page request to
// GET /vault/folders/{folderName}/media. Both fields are optional; unset
// pointers are omitted from the query string.
type ListVaultFolderMediaParams struct {
	Page *int
	Size *int
}

// ListVaultFolderMedia returns one page of the media stored in a vault folder.
//
// GET /vault/folders/{folderName}/media — scope: read:vault.
func (c *Client) ListVaultFolderMedia(
	ctx context.Context, folderName string, p ListVaultFolderMediaParams,
) (*VaultFolderMediaPage, error) {
	if folderName == "" {
		return nil, errors.New("fanvue: ListVaultFolderMedia requires a non-empty folder name")
	}
	path := "/vault/folders/" + url.PathEscape(folderName) + "/media"
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &VaultFolderMediaPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AttachMediaResult is the response from AttachMediaToVaultFolder. AddedCount is
// the number of media items newly attached to the folder.
type AttachMediaResult struct {
	AddedCount float64 `json:"addedCount"`
}

// AttachMediaToVaultFolder adds media to a vault folder. Body is the opaque
// request payload (mirroring the Python SDK's Mapping[str, Any] signature) and
// must be supplied — typically {"mediaUuids": ["..."]}.
//
// POST /vault/folders/{folderName}/media — scope: write:vault.
func (c *Client) AttachMediaToVaultFolder(
	ctx context.Context, folderName string, body RawBody,
) (*AttachMediaResult, error) {
	if folderName == "" {
		return nil, errors.New("fanvue: AttachMediaToVaultFolder requires a non-empty folder name")
	}
	if len(body) == 0 {
		return nil, errors.New("fanvue: AttachMediaToVaultFolder requires a non-empty body")
	}
	path := "/vault/folders/" + url.PathEscape(folderName) + "/media"
	out := &AttachMediaResult{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DetachMediaFromVaultFolder removes a single media item from a vault folder.
// The endpoint returns no body.
//
// DELETE /vault/folders/{folderName}/media/{mediaUuid} — scope: write:vault.
func (c *Client) DetachMediaFromVaultFolder(
	ctx context.Context, folderName, mediaUUID string,
) error {
	if folderName == "" {
		return errors.New("fanvue: DetachMediaFromVaultFolder requires a non-empty folder name")
	}
	if mediaUUID == "" {
		return errors.New("fanvue: DetachMediaFromVaultFolder requires a non-empty media UUID")
	}
	path := "/vault/folders/" + url.PathEscape(folderName) +
		"/media/" + url.PathEscape(mediaUUID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// Collection is a single content collection owned by the authenticated user. It
// is the shape returned by ListCollections (per item) and CreateCollection,
// which share the same payload. CreatedAt is nil when the API reports a JSON
// null.
type Collection struct {
	CreatedAt *string `json:"createdAt"`
	Label     string  `json:"label"`
	UUID      string  `json:"uuid"`
}

// CollectionsPage is one page of the authenticated user's content collections.
type CollectionsPage struct {
	Data       []Collection `json:"data"`
	Pagination Pagination   `json:"pagination"`
}

// ListCollections returns the authenticated user's content collections.
//
// GET /collections — scope: read:collection.
func (c *Client) ListCollections(ctx context.Context) (*CollectionsPage, error) {
	out := &CollectionsPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/collections", nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateCollection creates a content collection. Body is the opaque request
// payload (mirroring the Python SDK's Mapping[str, Any] signature) and must be
// supplied — typically {"label": "..."}.
//
// POST /collections — scope: write:collection.
func (c *Client) CreateCollection(ctx context.Context, body RawBody) (*Collection, error) {
	if len(body) == 0 {
		return nil, errors.New("fanvue: CreateCollection requires a non-empty body")
	}
	out := &Collection{}
	if err := c.doJSON(ctx, http.MethodPost, "/collections", nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// RenameCollection renames a content collection. Body is the opaque request
// payload (mirroring the Python SDK's Mapping[str, Any] signature) and must be
// supplied — typically {"label": "..."}. The endpoint returns no body.
//
// PATCH /collections/{uuid} — scope: write:collection.
func (c *Client) RenameCollection(ctx context.Context, collectionUUID string, body RawBody) error {
	if collectionUUID == "" {
		return errors.New("fanvue: RenameCollection requires a non-empty collection UUID")
	}
	if len(body) == 0 {
		return errors.New("fanvue: RenameCollection requires a non-empty body")
	}
	path := "/collections/" + url.PathEscape(collectionUUID)
	return c.doJSON(ctx, http.MethodPatch, path, nil, body, nil)
}

// DeleteCollection deletes a content collection by UUID. The endpoint returns no
// body.
//
// DELETE /collections/{uuid} — scope: write:collection.
func (c *Client) DeleteCollection(ctx context.Context, collectionUUID string) error {
	if collectionUUID == "" {
		return errors.New("fanvue: DeleteCollection requires a non-empty collection UUID")
	}
	path := "/collections/" + url.PathEscape(collectionUUID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// NotificationOriginator is the user who triggered a notification. It is nil
// (omitted) for system-generated notifications with no originating user.
type NotificationOriginator struct {
	AvatarURL   *string `json:"avatarUrl"`
	DisplayName string  `json:"displayName"`
	Handle      string  `json:"handle"`
	IsCreator   bool    `json:"isCreator"`
	UUID        string  `json:"uuid"`
}

// Notification is a single entry in the authenticated user's notifications feed.
//
// Data is the per-event payload, whose shape varies by EventType; it is exposed
// verbatim as RawBody (mirroring the Python SDK's dict[str, Any | None]) so
// callers can decode it into whatever shape the specific event documents.
// Originator is nil for system notifications with no originating user.
type Notification struct {
	CreatedAt    string                  `json:"createdAt"`
	Data         RawBody                 `json:"data"`
	EventType    int                     `json:"eventType"`
	IsRead       bool                    `json:"isRead"`
	Originator   *NotificationOriginator `json:"originator"`
	ReceiverUUID string                  `json:"receiverUuid"`
	UUID         string                  `json:"uuid"`
}

// NotificationsPage is one page of the authenticated user's notifications feed.
type NotificationsPage struct {
	Data       []Notification `json:"data"`
	Pagination Pagination     `json:"pagination"`
}

// ListNotificationsParams configures a page request to GET /notifications. All
// fields are optional; unset pointers are omitted from the query string.
// EventType, when set, filters the feed to a single numeric event type.
type ListNotificationsParams struct {
	Page      *int
	Size      *int
	EventType *int
}

// ListNotifications returns one page of the authenticated user's notifications
// feed.
//
// GET /notifications — scope: read:notification.
func (c *Client) ListNotifications(
	ctx context.Context, p ListNotificationsParams,
) (*NotificationsPage, error) {
	query := encodeQuery(map[string]any{
		"page":      p.Page,
		"size":      p.Size,
		"eventType": p.EventType,
	})
	out := &NotificationsPage{}
	if err := c.doJSON(ctx, http.MethodGet, "/notifications", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
