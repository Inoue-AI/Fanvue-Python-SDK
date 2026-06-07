package fanvue

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// This file implements the creator-scoped media surface: a managed creator's
// media list/detail, the S3 multipart upload session lifecycle, and the
// creator's vault folders + folder-media attach/detach. Every method here is an
// agency endpoint operating on a specific managed creator identified by
// creatorUserUUID, mirroring the self-scoped /media and /vault endpoints exactly
// in verb, parameters, defaults, pagination, and error mapping. Response shapes
// are byte-identical to their self-scoped counterparts, so the existing
// MediaItem, MediaPage, CreateUploadSessionResult, CompleteUploadSessionResult,
// VaultFolder, VaultFoldersPage, VaultFolderMediaItem, VaultFolderMediaPage, and
// AttachMediaResult types are reused.

// ListCreatorMediaParams configures a page request to
// GET /creators/{creatorUserUuid}/media. All fields are optional; unset pointers
// and empty slices are omitted from the query string. It mirrors the self-scoped
// ListUserMediaParams field-for-field.
type ListCreatorMediaParams struct {
	Page        *int
	Size        *int
	MediaType   *MediaType
	FolderName  *string
	Usage       *MediaUsage
	PurchasedBy *string
	Status      []MediaStatus
	Variants    []MediaVariantType
}

// GetCreatorMedia returns one page of a managed creator's media. Non-finalised
// items carry only UUID and Status; finalised items include full details and the
// requested variant URLs. The response shape is identical to the self-scoped
// GET /media list, so it reuses MediaPage.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/media — scopes: read:media, read:creator.
func (c *Client) GetCreatorMedia(
	ctx context.Context, creatorUserUUID string, p ListCreatorMediaParams,
) (*MediaPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorMedia requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/media"
	query := encodeQuery(map[string]any{
		"page":        p.Page,
		"size":        p.Size,
		"mediaType":   mediaTypePtrToString(p.MediaType),
		"folderName":  p.FolderName,
		"usage":       mediaUsagePtrToString(p.Usage),
		"purchasedBy": p.PurchasedBy,
		"status":      mediaStatusesToStrings(p.Status),
		"variants":    mediaVariantsToStrings(p.Variants),
	})
	out := &MediaPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCreatorMediaByUUIDParams configures the optional query parameters for
// GetCreatorMediaByUUID. Media URLs are only returned for the variants
// requested. It mirrors the self-scoped GetUserMediaByUUIDParams.
type GetCreatorMediaByUUIDParams struct {
	PurchasedBy *string
	Variants    []MediaVariantType
}

// GetCreatorMediaByUUID fetches a single media item owned by a managed creator by
// its UUID. Without Variants the variants field is empty and no media URLs are
// returned. The response shape is identical to the self-scoped GET /media/{uuid},
// so it reuses MediaItem.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/media/{uuid} — scopes: read:media, read:creator.
func (c *Client) GetCreatorMediaByUUID(
	ctx context.Context, creatorUserUUID, mediaUUID string, p GetCreatorMediaByUUIDParams,
) (*MediaItem, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorMediaByUUID requires a non-empty creator user UUID")
	}
	if mediaUUID == "" {
		return nil, errors.New("fanvue: GetCreatorMediaByUUID requires a non-empty media UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/media/" + url.PathEscape(mediaUUID)
	query := encodeQuery(map[string]any{
		"purchasedBy": p.PurchasedBy,
		"variants":    mediaVariantsToStrings(p.Variants),
	})
	out := &MediaItem{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateCreatorUploadSession creates a media record and starts an S3 multipart
// upload session on behalf of a managed creator. Body is the opaque request
// payload (mirroring the Python SDK's Mapping[str, Any] signature) and must be
// supplied — typically describing the media type, file name, and part count. The
// response shape is identical to the self-scoped POST /media/uploads, so it
// reuses CreateUploadSessionResult.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// POST /creators/{creatorUserUuid}/media/uploads — scopes: write:media, write:creator.
func (c *Client) CreateCreatorUploadSession(
	ctx context.Context, creatorUserUUID string, body RawBody,
) (*CreateUploadSessionResult, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: CreateCreatorUploadSession requires a non-empty creator user UUID")
	}
	if len(body) == 0 {
		return nil, errors.New("fanvue: CreateCreatorUploadSession requires a non-empty body")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/media/uploads"
	out := &CreateUploadSessionResult{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CompleteCreatorUploadSession completes the multipart upload in S3 and
// transitions a managed creator's media to processing. Media URLs become
// available once processing completes. Body is the opaque request payload
// (mirroring the Python SDK's Mapping[str, Any] signature) and must be supplied —
// typically the list of uploaded part ETags. The response shape is identical to
// the self-scoped PATCH /media/uploads/{uploadId}, so it reuses
// CompleteUploadSessionResult.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// PATCH /creators/{creatorUserUuid}/media/uploads/{uploadId} — scopes: write:media, write:creator.
func (c *Client) CompleteCreatorUploadSession(
	ctx context.Context, creatorUserUUID, uploadID string, body RawBody,
) (*CompleteUploadSessionResult, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: CompleteCreatorUploadSession requires a non-empty creator user UUID")
	}
	if uploadID == "" {
		return nil, errors.New("fanvue: CompleteCreatorUploadSession requires a non-empty upload ID")
	}
	if len(body) == 0 {
		return nil, errors.New("fanvue: CompleteCreatorUploadSession requires a non-empty body")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/media/uploads/" + url.PathEscape(uploadID)
	out := &CompleteUploadSessionResult{}
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCreatorUploadPartURL returns a presigned URL for uploading a specific part
// of a managed creator's media multipart upload session. partNumber is the
// 1-based index of the part.
//
// The Python SDK discards this endpoint's body and returns None; to avoid
// silently losing the signed URL while preserving identical endpoint, verb,
// parameters, and error mapping, the Go SDK returns the raw JSON body verbatim —
// mirroring the self-scoped GetUploadPartURL exactly. Callers can unmarshal it
// into whatever shape the API documents.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/media/uploads/{uploadId}/parts/{partNumber}/url
// — scopes: write:media, write:creator.
func (c *Client) GetCreatorUploadPartURL(
	ctx context.Context, creatorUserUUID, uploadID string, partNumber int,
) (RawBody, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorUploadPartURL requires a non-empty creator user UUID")
	}
	if uploadID == "" {
		return nil, errors.New("fanvue: GetCreatorUploadPartURL requires a non-empty upload ID")
	}
	if partNumber < 1 {
		return nil, fmt.Errorf("fanvue: GetCreatorUploadPartURL requires a positive part number, got %d", partNumber)
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/media/uploads/" + url.PathEscape(uploadID) +
		"/parts/" + url.PathEscape(strconv.Itoa(partNumber)) + "/url"
	out := RawBody{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListCreatorVaultFolders returns a managed creator's vault folders. The response
// shape is identical to the self-scoped GET /vault/folders, so it reuses
// VaultFoldersPage.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/vault/folders — scopes: read:vault, read:creator.
func (c *Client) ListCreatorVaultFolders(
	ctx context.Context, creatorUserUUID string,
) (*VaultFoldersPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: ListCreatorVaultFolders requires a non-empty creator user UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/vault/folders"
	out := &VaultFoldersPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateCreatorVaultFolder creates a vault folder for a managed creator. Body is
// the opaque request payload (mirroring the Python SDK's Mapping[str, Any]
// signature) and must be supplied — typically {"name": "..."}. The response shape
// is identical to the self-scoped POST /vault/folders, so it reuses VaultFolder.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// POST /creators/{creatorUserUuid}/vault/folders — scopes: write:vault, write:creator.
func (c *Client) CreateCreatorVaultFolder(
	ctx context.Context, creatorUserUUID string, body RawBody,
) (*VaultFolder, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: CreateCreatorVaultFolder requires a non-empty creator user UUID")
	}
	if len(body) == 0 {
		return nil, errors.New("fanvue: CreateCreatorVaultFolder requires a non-empty body")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) + "/vault/folders"
	out := &VaultFolder{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCreatorVaultFolder fetches a single vault folder of a managed creator by its
// name. The response shape is identical to the self-scoped
// GET /vault/folders/{folderName}, so it reuses VaultFolder.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/vault/folders/{folderName} — scopes: read:vault, read:creator.
func (c *Client) GetCreatorVaultFolder(
	ctx context.Context, creatorUserUUID, folderName string,
) (*VaultFolder, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: GetCreatorVaultFolder requires a non-empty creator user UUID")
	}
	if folderName == "" {
		return nil, errors.New("fanvue: GetCreatorVaultFolder requires a non-empty folder name")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/vault/folders/" + url.PathEscape(folderName)
	out := &VaultFolder{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// RenameCreatorVaultFolder renames a managed creator's vault folder. Body is the
// opaque request payload (mirroring the Python SDK's Mapping[str, Any] signature)
// and must be supplied — typically {"name": "..."}. The endpoint returns no body.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// PATCH /creators/{creatorUserUuid}/vault/folders/{folderName} — scopes: write:vault, write:creator.
func (c *Client) RenameCreatorVaultFolder(
	ctx context.Context, creatorUserUUID, folderName string, body RawBody,
) error {
	if creatorUserUUID == "" {
		return errors.New("fanvue: RenameCreatorVaultFolder requires a non-empty creator user UUID")
	}
	if folderName == "" {
		return errors.New("fanvue: RenameCreatorVaultFolder requires a non-empty folder name")
	}
	if len(body) == 0 {
		return errors.New("fanvue: RenameCreatorVaultFolder requires a non-empty body")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/vault/folders/" + url.PathEscape(folderName)
	return c.doJSON(ctx, http.MethodPatch, path, nil, body, nil)
}

// DeleteCreatorVaultFolder deletes a managed creator's vault folder by name. The
// endpoint returns no body.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// DELETE /creators/{creatorUserUuid}/vault/folders/{folderName} — scopes: write:vault, write:creator.
func (c *Client) DeleteCreatorVaultFolder(
	ctx context.Context, creatorUserUUID, folderName string,
) error {
	if creatorUserUUID == "" {
		return errors.New("fanvue: DeleteCreatorVaultFolder requires a non-empty creator user UUID")
	}
	if folderName == "" {
		return errors.New("fanvue: DeleteCreatorVaultFolder requires a non-empty folder name")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/vault/folders/" + url.PathEscape(folderName)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// ListCreatorVaultFolderMediaParams configures a page request to
// GET /creators/{creatorUserUuid}/vault/folders/{folderName}/media. Both fields
// are optional; unset pointers are omitted from the query string. It mirrors the
// self-scoped ListVaultFolderMediaParams.
type ListCreatorVaultFolderMediaParams struct {
	Page *int
	Size *int
}

// ListCreatorVaultFolderMedia returns one page of the media stored in a managed
// creator's vault folder. The response shape is identical to the self-scoped
// GET /vault/folders/{folderName}/media, so it reuses VaultFolderMediaPage.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// GET /creators/{creatorUserUuid}/vault/folders/{folderName}/media — scopes: read:vault, read:creator.
func (c *Client) ListCreatorVaultFolderMedia(
	ctx context.Context, creatorUserUUID, folderName string, p ListCreatorVaultFolderMediaParams,
) (*VaultFolderMediaPage, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: ListCreatorVaultFolderMedia requires a non-empty creator user UUID")
	}
	if folderName == "" {
		return nil, errors.New("fanvue: ListCreatorVaultFolderMedia requires a non-empty folder name")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/vault/folders/" + url.PathEscape(folderName) + "/media"
	query := encodeQuery(map[string]any{"page": p.Page, "size": p.Size})
	out := &VaultFolderMediaPage{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// AttachCreatorVaultMedia adds media to a managed creator's vault folder. Body is
// the opaque request payload (mirroring the Python SDK's Mapping[str, Any]
// signature) and must be supplied — typically {"mediaUuids": ["..."]}. The
// response shape is identical to the self-scoped POST
// /vault/folders/{folderName}/media, so it reuses AttachMediaResult.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// POST /creators/{creatorUserUuid}/vault/folders/{folderName}/media — scopes: write:vault, write:creator.
func (c *Client) AttachCreatorVaultMedia(
	ctx context.Context, creatorUserUUID, folderName string, body RawBody,
) (*AttachMediaResult, error) {
	if creatorUserUUID == "" {
		return nil, errors.New("fanvue: AttachCreatorVaultMedia requires a non-empty creator user UUID")
	}
	if folderName == "" {
		return nil, errors.New("fanvue: AttachCreatorVaultMedia requires a non-empty folder name")
	}
	if len(body) == 0 {
		return nil, errors.New("fanvue: AttachCreatorVaultMedia requires a non-empty body")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/vault/folders/" + url.PathEscape(folderName) + "/media"
	out := &AttachMediaResult{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DetachCreatorVaultMedia removes a single media item from a managed creator's
// vault folder. The endpoint returns no body.
//
// This is an agency endpoint: it operates on a specific creator the
// authenticated agency manages, identified by creatorUserUUID.
//
// DELETE /creators/{creatorUserUuid}/vault/folders/{folderName}/media/{mediaUuid}
// — scopes: write:vault, write:creator.
func (c *Client) DetachCreatorVaultMedia(
	ctx context.Context, creatorUserUUID, folderName, mediaUUID string,
) error {
	if creatorUserUUID == "" {
		return errors.New("fanvue: DetachCreatorVaultMedia requires a non-empty creator user UUID")
	}
	if folderName == "" {
		return errors.New("fanvue: DetachCreatorVaultMedia requires a non-empty folder name")
	}
	if mediaUUID == "" {
		return errors.New("fanvue: DetachCreatorVaultMedia requires a non-empty media UUID")
	}
	path := "/creators/" + url.PathEscape(creatorUserUUID) +
		"/vault/folders/" + url.PathEscape(folderName) +
		"/media/" + url.PathEscape(mediaUUID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}
