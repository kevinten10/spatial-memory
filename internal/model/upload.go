package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MediaTypeSizeLimits defines max file sizes per media type.
var MediaTypeSizeLimits = map[MediaType]int64{
	MediaTypePhoto: 20 * 1024 * 1024, // 20MB
	MediaTypeVideo: 100 * 1024 * 1024, // 100MB
	MediaTypeVoice: 10 * 1024 * 1024, // 10MB
}

// AllowedMimeTypes maps media types to allowed MIME types.
var AllowedMimeTypes = map[MediaType][]string{
	MediaTypePhoto: {"image/jpeg", "image/png", "image/heic", "image/webp"},
	MediaTypeVideo: {"video/mp4", "video/quicktime", "video/webm"},
	MediaTypeVoice: {"audio/mpeg", "audio/mp4", "audio/wav", "audio/webm"},
}

// UploadRequest is the request to get a pre-signed upload URL.
type UploadRequest struct {
	MemoryID    int64     `json:"memory_id" binding:"required"`
	MediaType   MediaType `json:"media_type" binding:"min=0,max=2"`
	ContentType string    `json:"content_type" binding:"required"`
	FileSize    int64     `json:"file_size" binding:"required"`
	ContentHash string    `json:"content_hash" binding:"required,len=64"` // SHA-256 hex
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Duration    int       `json:"duration"` // seconds, for video/voice
}

// UploadResponse contains the pre-signed URL and storage key.
type UploadResponse struct {
	UploadURL   string `json:"upload_url"`
	StorageKey  string `json:"storage_key"`
	ExpiresIn   int    `json:"expires_in"` // seconds
	ContentHash string `json:"content_hash"`
	IsDuplicate bool   `json:"is_duplicate"` // if true, file already exists
	ExistingURL string `json:"existing_url,omitempty"`
}

// ConfirmUploadRequest confirms a file has been uploaded.
type ConfirmUploadRequest struct {
	StorageKey  string `json:"storage_key" binding:"required"`
	ContentHash string `json:"content_hash" binding:"required,len=64"`
}

// GenerateStorageKey creates a unique storage key for a file.
// Pattern: memories/<user_id>/<memory_id>/<type>/<uuid>.<ext>
func GenerateStorageKey(userID, memoryID int64, mediaType MediaType, contentType string) string {
	var typeDir string
	switch mediaType {
	case MediaTypePhoto:
		typeDir = "photos"
	case MediaTypeVideo:
		typeDir = "videos"
	case MediaTypeVoice:
		typeDir = "voice"
	default:
		typeDir = "files"
	}

	ext := extFromContentType(contentType)
	fileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	return filepath.Join(
		"memories",
		fmt.Sprintf("%d", userID),
		fmt.Sprintf("%d", memoryID),
		typeDir,
		fileName,
	)
}

// HashContent generates SHA-256 hash of content.
func HashContent(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}

// IsAllowedMimeType checks if the MIME type is allowed for the media type.
func IsAllowedMimeType(mediaType MediaType, contentType string) bool {
	allowed, ok := AllowedMimeTypes[mediaType]
	if !ok {
		return false
	}

	contentType = strings.ToLower(contentType)
	for _, mime := range allowed {
		if mime == contentType {
			return true
		}
	}
	return false
}

// IsAllowedFileSize checks if the file size is within limits.
func IsAllowedFileSize(mediaType MediaType, size int64) bool {
	limit, ok := MediaTypeSizeLimits[mediaType]
	if !ok {
		return false
	}
	return size <= limit
}

func extFromContentType(contentType string) string {
	parts := strings.Split(contentType, "/")
	if len(parts) != 2 {
		return ""
	}

	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/heic":
		return ".heic"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/quicktime":
		return ".mov"
	case "video/webm":
		return ".webm"
	case "audio/mpeg":
		return ".mp3"
	case "audio/mp4":
		return ".m4a"
	case "audio/wav":
		return ".wav"
	case "audio/webm":
		return ".weba"
	default:
		return ""
	}
}

// PreSignedURLExpiry is the duration for which pre-signed URLs are valid.
const PreSignedURLExpiry = 15 * time.Minute
