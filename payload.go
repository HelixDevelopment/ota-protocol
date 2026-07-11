package otaprotocol

import (
	"fmt"
	"strconv"
)

// AOSP update_engine payload_properties header keys passed straight through to
// applyPayload's keyValuePairHeaders (PayloadProperties schema).
const (
	HeaderFileHash     = "FILE_HASH"
	HeaderFileSize     = "FILE_SIZE"
	HeaderMetadataHash = "METADATA_HASH"
	HeaderMetadataSize = "METADATA_SIZE"
)

// ErrMissingPayloadHeader indicates a required payload_properties header was
// absent when parsing a header map.
var ErrMissingPayloadHeader = fmt.Errorf("otaprotocol: missing payload_properties header")

// PayloadProperties is the typed form of the four mandatory AOSP
// payload_properties headers. The optional headers (SWITCH_SLOT_ON_REBOOT,
// RUN_POST_INSTALL, DISABLE_DOWNLOAD_RESUME) are intentionally omitted (their
// Android 15 behavior is unverified).
type PayloadProperties struct {
	FileHash     string `json:"FILE_HASH"`
	FileSize     int64  `json:"FILE_SIZE"`
	MetadataHash string `json:"METADATA_HASH"`
	MetadataSize int64  `json:"METADATA_SIZE"`
}

// Headers renders the PayloadProperties into the string key/value map that
// update_engine.applyPayload expects. Integer sizes are rendered as base-10
// strings.
func (p PayloadProperties) Headers() map[string]string {
	return map[string]string{
		HeaderFileHash:     p.FileHash,
		HeaderFileSize:     strconv.FormatInt(p.FileSize, 10),
		HeaderMetadataHash: p.MetadataHash,
		HeaderMetadataSize: strconv.FormatInt(p.MetadataSize, 10),
	}
}

// Validate reports whether all four headers carry usable values: non-empty
// hashes and non-negative sizes.
func (p PayloadProperties) Validate() error {
	if p.FileHash == "" {
		return fmt.Errorf("%w: %s", ErrMissingField, HeaderFileHash)
	}
	if p.MetadataHash == "" {
		return fmt.Errorf("%w: %s", ErrMissingField, HeaderMetadataHash)
	}
	if p.FileSize < 0 {
		return fmt.Errorf("%w: %s must be >= 0, got %d", ErrInvalidValue, HeaderFileSize, p.FileSize)
	}
	if p.MetadataSize < 0 {
		return fmt.Errorf("%w: %s must be >= 0, got %d", ErrInvalidValue, HeaderMetadataSize, p.MetadataSize)
	}
	return nil
}

// ParsePayloadProperties parses an update_engine header map back into a typed
// PayloadProperties. All four mandatory keys must be present; the size values
// must be valid base-10 integers.
func ParsePayloadProperties(h map[string]string) (PayloadProperties, error) {
	var p PayloadProperties

	fileHash, ok := h[HeaderFileHash]
	if !ok {
		return p, fmt.Errorf("%w: %s", ErrMissingPayloadHeader, HeaderFileHash)
	}
	metaHash, ok := h[HeaderMetadataHash]
	if !ok {
		return p, fmt.Errorf("%w: %s", ErrMissingPayloadHeader, HeaderMetadataHash)
	}
	fileSize, err := parseHeaderInt(h, HeaderFileSize)
	if err != nil {
		return p, err
	}
	metaSize, err := parseHeaderInt(h, HeaderMetadataSize)
	if err != nil {
		return p, err
	}
	if fileSize < 0 {
		return p, fmt.Errorf("%w: %s must be >= 0, got %d", ErrInvalidValue, HeaderFileSize, fileSize)
	}
	if metaSize < 0 {
		return p, fmt.Errorf("%w: %s must be >= 0, got %d", ErrInvalidValue, HeaderMetadataSize, metaSize)
	}

	p = PayloadProperties{
		FileHash:     fileHash,
		FileSize:     fileSize,
		MetadataHash: metaHash,
		MetadataSize: metaSize,
	}
	return p, nil
}

// parseHeaderInt reads a required base-10 int64 header value.
func parseHeaderInt(h map[string]string, key string) (int64, error) {
	raw, ok := h[key]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrMissingPayloadHeader, key)
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s = %q: %v", ErrInvalidValue, key, raw, err)
	}
	return n, nil
}
