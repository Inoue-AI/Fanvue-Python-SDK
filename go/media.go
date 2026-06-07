package fanvue

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// MediaType enumerates the kinds of media the authenticated user can own.
type MediaType string

const (
	// MediaTypeImage is a still image.
	MediaTypeImage MediaType = "image"
	// MediaTypeVideo is a video clip.
	MediaTypeVideo MediaType = "video"
	// MediaTypeAudio is an audio clip.
	MediaTypeAudio MediaType = "audio"
	// MediaTypeDocument is a document attachment.
	MediaTypeDocument MediaType = "document"
)

// MediaStatus enumerates the lifecycle states of a media item. Only FINALISED
// ("ready") media expose full details and variant URLs.
type MediaStatus string

const (
	// MediaStatusCreated is a freshly created media record awaiting upload.
	MediaStatusCreated MediaStatus = "created"
	// MediaStatusProcessing is media being transcoded after a completed upload.
	MediaStatusProcessing MediaStatus = "processing"
	// MediaStatusReady is finalised media with variant URLs available.
	MediaStatusReady MediaStatus = "ready"
	// MediaStatusError is media that failed processing.
	MediaStatusError MediaStatus = "error"
)

// MediaUsage enumerates the audiences a media item can be filtered by when
// listing the authenticated user's media.
type MediaUsage string

const (
	// MediaUsageSubscribers filters to media used in subscriber-only content.
	MediaUsageSubscribers MediaUsage = "subscribers"
	// MediaUsageFollowers filters to media used in follower-visible content.
	MediaUsageFollowers MediaUsage = "followers"
	// MediaUsagePPV filters to media used in pay-per-view content.
	MediaUsagePPV MediaUsage = "ppv"
	// MediaUsageMassMessages filters to media used in mass messages.
	MediaUsageMassMessages MediaUsage = "mass_messages"
)

// MediaTags holds the structured AI content tags Fanvue attaches to finalised
// media when the owning creator has AI content tagging enabled. It is nil
// (omitted) otherwise.
type MediaTags struct {
	BodyParts     []string  `json:"bodyParts"`
	BodyType      []string  `json:"bodyType"`
	Description   *string   `json:"description"`
	HairColor     []string  `json:"hairColor"`
	ImportantTags []string  `json:"importantTags"`
	IsNsfw        bool      `json:"isNsfw"`
	MediaType     MediaType `json:"mediaType"`
	NsfwCategory  []string  `json:"nsfwCategory"`
	OtherTags     []string  `json:"otherTags"`
	People        []string  `json:"people"`
	Position      []string  `json:"position"`
	Setting       []string  `json:"setting"`
	SexActs       []string  `json:"sexActs"`
	SexObjects    []string  `json:"sexObjects"`
	SkinColor     []string  `json:"skinColor"`
	Tags          []string  `json:"tags"`
}

// MediaResourceVariant is one rendered variant of a media-resource item.
// Nullable dimensions use pointers so a JSON null is distinguishable from a zero
// value; URL is omitted (nil) when the variant URL is not available. This mirrors
// the Python *VariantsItem models, which (unlike the chat MediaVariant) include
// a per-variant uuid.
type MediaResourceVariant struct {
	DisplayPosition float64          `json:"displayPosition"`
	Height          *float64         `json:"height"`
	LengthMs        *float64         `json:"lengthMs"`
	URL             *string          `json:"url,omitempty"`
	UUID            string           `json:"uuid"`
	VariantType     MediaVariantType `json:"variantType"`
	Width           *float64         `json:"width"`
}

// MediaItem is a single media item owned by the authenticated user.
//
// The Fanvue API returns a discriminated shape: for media whose status is not
// FINALISED only UUID and Status are populated; for FINALISED media every field
// is populated (subject to the requested variants). Because the non-finalised
// shape is a strict subset of the finalised one, both are represented losslessly
// by this single struct with pointers on the finalised-only fields — matching the
// Python SDK's Option1 | Option2 union without forcing callers to type-switch.
type MediaItem struct {
	UUID   string      `json:"uuid"`
	Status MediaStatus `json:"status"`

	// Finalised-only fields. These are nil/omitted for non-FINALISED media.
	Caption          *string                `json:"caption,omitempty"`
	CreatedAt        *string                `json:"createdAt,omitempty"`
	Description      *string                `json:"description,omitempty"`
	MediaType        *MediaType             `json:"mediaType,omitempty"`
	Name             *string                `json:"name,omitempty"`
	PurchasedByFan   *bool                  `json:"purchasedByFan,omitempty"`
	RecommendedPrice *float64               `json:"recommendedPrice,omitempty"`
	Tags             *MediaTags             `json:"tags,omitempty"`
	URL              *string                `json:"url,omitempty"`
	Variants         []MediaResourceVariant `json:"variants,omitempty"`
}

// MediaPage is one page of the authenticated user's media list.
type MediaPage struct {
	Data       []MediaItem `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// ListUserMediaParams configures a page request to GET /media. All fields are
// optional; unset pointers and empty slices are omitted from the query string.
type ListUserMediaParams struct {
	Page        *int
	Size        *int
	MediaType   *MediaType
	FolderName  *string
	Usage       *MediaUsage
	PurchasedBy *string
	Status      []MediaStatus
	Variants    []MediaVariantType
}

// GetUserMedia returns one page of the authenticated user's media. Non-finalised
// items carry only UUID and Status; finalised items include full details and the
// requested variant URLs.
//
// GET /media — scope: read:media.
func (c *Client) GetUserMedia(ctx context.Context, p ListUserMediaParams) (*MediaPage, error) {
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
	if err := c.doJSON(ctx, http.MethodGet, "/media", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetUserMediaByUUIDParams configures the optional query parameters for
// GetUserMediaByUUID. Media URLs are only returned for the variants requested.
type GetUserMediaByUUIDParams struct {
	PurchasedBy *string
	Variants    []MediaVariantType
}

// GetUserMediaByUUID fetches a single media item owned by the authenticated user
// by its UUID. Without Variants the variants field is empty and no media URLs are
// returned.
//
// GET /media/{uuid} — scope: read:media.
func (c *Client) GetUserMediaByUUID(
	ctx context.Context, mediaUUID string, p GetUserMediaByUUIDParams,
) (*MediaItem, error) {
	if mediaUUID == "" {
		return nil, errors.New("fanvue: GetUserMediaByUUID requires a non-empty media UUID")
	}
	path := "/media/" + url.PathEscape(mediaUUID)
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

// BulkMediaError is a per-UUID failure entry returned by GetBulkMedia when a
// requested media UUID could not be resolved.
type BulkMediaError struct {
	// Code is "NOT_FOUND" or "INTERNAL".
	Code      string `json:"code"`
	MediaUUID string `json:"mediaUuid"`
	Message   string `json:"message"`
}

// BulkMediaResult is the response from GetBulkMedia. Results maps each requested
// media UUID to its resolved item, or to a JSON null (nil pointer) when that UUID
// failed; the parallel Errors slice carries the failure detail.
type BulkMediaResult struct {
	Errors  []BulkMediaError      `json:"errors"`
	Results map[string]*MediaItem `json:"results"`
}

// GetBulkMediaParams configures a GET /media/bulk request. MediaUUIDs is a
// required comma-separated list of media UUIDs (maximum 20 per request);
// Variants, when set, is a comma-separated list of variant types to include.
// Both mirror the Python SDK's plain-string query parameters verbatim.
type GetBulkMediaParams struct {
	MediaUUIDs string
	Variants   *string
}

// GetBulkMedia returns multiple media items in a single request, keyed by UUID.
// A maximum of 20 media UUIDs may be requested per call.
//
// GET /media/bulk — scope: read:media.
func (c *Client) GetBulkMedia(ctx context.Context, p GetBulkMediaParams) (*BulkMediaResult, error) {
	if p.MediaUUIDs == "" {
		return nil, errors.New("fanvue: GetBulkMedia requires non-empty MediaUUIDs")
	}
	query := encodeQuery(map[string]any{
		"mediaUuids": p.MediaUUIDs,
		"variants":   p.Variants,
	})
	out := &BulkMediaResult{}
	if err := c.doJSON(ctx, http.MethodGet, "/media/bulk", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetEntitledMediaParams configures a GET /media/{uuid}/entitled request.
// ConsumerID is required and identifies the consumer who must have been granted
// access via a prior GrantMedia call; Variants selects which variant URLs to
// include.
type GetEntitledMediaParams struct {
	ConsumerID string
	Variants   []MediaVariantType
}

// GetEntitledMedia returns a media item with signed variant URLs, but only if the
// specified consumer has been granted access to it. Media URLs are only available
// through the requested variants.
//
// GET /media/{uuid}/entitled — scope: read:media.
func (c *Client) GetEntitledMedia(
	ctx context.Context, mediaUUID string, p GetEntitledMediaParams,
) (*MediaItem, error) {
	if mediaUUID == "" {
		return nil, errors.New("fanvue: GetEntitledMedia requires a non-empty media UUID")
	}
	if p.ConsumerID == "" {
		return nil, errors.New("fanvue: GetEntitledMedia requires a non-empty ConsumerID")
	}
	path := "/media/" + url.PathEscape(mediaUUID) + "/entitled"
	query := encodeQuery(map[string]any{
		"consumerId": p.ConsumerID,
		"variants":   mediaVariantsToStrings(p.Variants),
	})
	out := &MediaItem{}
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// MediaLinkPurchaseStatus reports whether the authenticated user has a paid
// invoice for a given media link.
type MediaLinkPurchaseStatus struct {
	Purchased bool `json:"purchased"`
}

// GetMediaLinkPurchaseStatus reports whether the authenticated user has purchased
// the media link identified by linkUUID. The API returns 404 (mapped to a *Error
// with IsNotFound) when no media link with that UUID exists.
//
// GET /media/links/{uuid}/purchased — scope: read:media.
func (c *Client) GetMediaLinkPurchaseStatus(
	ctx context.Context, linkUUID string,
) (*MediaLinkPurchaseStatus, error) {
	if linkUUID == "" {
		return nil, errors.New("fanvue: GetMediaLinkPurchaseStatus requires a non-empty media link UUID")
	}
	path := "/media/links/" + url.PathEscape(linkUUID) + "/purchased"
	out := &MediaLinkPurchaseStatus{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GrantMediaResult is the response from GrantMedia. The grant is idempotent:
// repeated calls with the same parameters return the existing entitlement.
type GrantMediaResult struct {
	EntitlementID string `json:"entitlementId"`
	// Status is always "granted".
	Status string `json:"status"`
}

// GrantMedia grants a consumer access to a media item owned by the authenticated
// creator. Body is the opaque request payload (mirroring the Python SDK's
// Mapping[str, Any] signature) and must be supplied — typically
// {"consumerId": "..."}.
//
// POST /media/{uuid}/grant — scope: write:media.
func (c *Client) GrantMedia(
	ctx context.Context, mediaUUID string, body RawBody,
) (*GrantMediaResult, error) {
	if mediaUUID == "" {
		return nil, errors.New("fanvue: GrantMedia requires a non-empty media UUID")
	}
	if len(body) == 0 {
		return nil, errors.New("fanvue: GrantMedia requires a non-empty body")
	}
	path := "/media/" + url.PathEscape(mediaUUID) + "/grant"
	out := &GrantMediaResult{}
	if err := c.doJSON(ctx, http.MethodPost, path, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateUploadSessionResult is the response from CreateUploadSession. It carries
// the new media record UUID and the S3 multipart upload session identifier used
// to request per-part signed URLs and to complete the session.
type CreateUploadSessionResult struct {
	MediaUUID string `json:"mediaUuid"`
	UploadID  string `json:"uploadId"`
}

// CreateUploadSession creates a media record and starts an S3 multipart upload
// session. Body is the opaque request payload (mirroring the Python SDK's
// Mapping[str, Any] signature) and must be supplied — typically describing the
// media type, file name, and part count.
//
// POST /media/uploads — scope: write:media.
func (c *Client) CreateUploadSession(
	ctx context.Context, body RawBody,
) (*CreateUploadSessionResult, error) {
	if len(body) == 0 {
		return nil, errors.New("fanvue: CreateUploadSession requires a non-empty body")
	}
	out := &CreateUploadSessionResult{}
	if err := c.doJSON(ctx, http.MethodPost, "/media/uploads", nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CompleteUploadSessionResult is the response from CompleteUploadSession. Status
// reflects the media lifecycle state after the multipart upload is finalised
// (typically "processing").
type CompleteUploadSessionResult struct {
	Status MediaStatus `json:"status"`
}

// CompleteUploadSession completes the multipart upload in S3 and transitions the
// media to processing. Media URLs become available once processing completes.
// Body is the opaque request payload (mirroring the Python SDK's
// Mapping[str, Any] signature) and must be supplied — typically the list of
// uploaded part ETags.
//
// PATCH /media/uploads/{uploadId} — scope: write:media.
func (c *Client) CompleteUploadSession(
	ctx context.Context, uploadID string, body RawBody,
) (*CompleteUploadSessionResult, error) {
	if uploadID == "" {
		return nil, errors.New("fanvue: CompleteUploadSession requires a non-empty upload ID")
	}
	if len(body) == 0 {
		return nil, errors.New("fanvue: CompleteUploadSession requires a non-empty body")
	}
	path := "/media/uploads/" + url.PathEscape(uploadID)
	out := &CompleteUploadSessionResult{}
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetUploadPartURL returns a presigned URL for uploading a specific part of a
// media multipart upload session. partNumber is the 1-based index of the part.
//
// The Python SDK discards this endpoint's body and returns None; to avoid
// silently losing the signed URL while preserving identical endpoint, verb,
// parameters, and error mapping, the Go SDK returns the raw JSON body verbatim.
// Callers can unmarshal it into whatever shape the API documents.
//
// GET /media/uploads/{uploadId}/parts/{partNumber}/url — scope: write:media.
func (c *Client) GetUploadPartURL(
	ctx context.Context, uploadID string, partNumber int,
) (RawBody, error) {
	if uploadID == "" {
		return nil, errors.New("fanvue: GetUploadPartURL requires a non-empty upload ID")
	}
	if partNumber < 1 {
		return nil, fmt.Errorf("fanvue: GetUploadPartURL requires a positive part number, got %d", partNumber)
	}
	path := "/media/uploads/" + url.PathEscape(uploadID) +
		"/parts/" + url.PathEscape(strconv.Itoa(partNumber)) + "/url"
	out := RawBody{}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// mediaTypePtrToString converts an optional MediaType into the *string form
// encodeQuery expects, returning nil when unset so the parameter is omitted.
func mediaTypePtrToString(t *MediaType) *string {
	if t == nil {
		return nil
	}
	s := string(*t)
	return &s
}

// mediaUsagePtrToString converts an optional MediaUsage into the *string form
// encodeQuery expects, returning nil when unset so the parameter is omitted.
func mediaUsagePtrToString(u *MediaUsage) *string {
	if u == nil {
		return nil
	}
	s := string(*u)
	return &s
}

// mediaStatusesToStrings flattens a slice of MediaStatus into the []string form
// encodeQuery renders as repeated query keys. It returns nil for an empty input
// so the parameter is omitted entirely.
func mediaStatusesToStrings(statuses []MediaStatus) []string {
	if len(statuses) == 0 {
		return nil
	}
	out := make([]string, len(statuses))
	for i, s := range statuses {
		out[i] = string(s)
	}
	return out
}

// mediaVariantsToStrings flattens a slice of MediaVariantType into the []string
// form encodeQuery renders as repeated query keys. It returns nil for an empty
// input so the parameter is omitted entirely.
func mediaVariantsToStrings(variants []MediaVariantType) []string {
	if len(variants) == 0 {
		return nil
	}
	out := make([]string, len(variants))
	for i, v := range variants {
		out[i] = string(v)
	}
	return out
}
