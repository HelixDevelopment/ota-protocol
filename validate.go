package otaprotocol

import (
	"errors"
	"fmt"
	"math"
)

// Sentinel errors returned (wrapped) by the validation helpers. Callers can
// match them with errors.Is.
var (
	// ErrInvalidEnum indicates an enum value outside its allowed set.
	ErrInvalidEnum = errors.New("otaprotocol: invalid enum value")
	// ErrInvalidSHA256 indicates a string that is not a 64-char lowercase hex
	// SHA-256 digest.
	ErrInvalidSHA256 = errors.New("otaprotocol: invalid sha256 (want 64 lowercase hex chars)")
	// ErrMissingField indicates a required field was empty/zero.
	ErrMissingField = errors.New("otaprotocol: missing required field")
	// ErrInvalidValue indicates a present field whose value is out of range or
	// otherwise malformed.
	ErrInvalidValue = errors.New("otaprotocol: invalid field value")
)

// ValidateSHA256 reports whether s is a well-formed SHA-256 digest: exactly 64
// characters, all lowercase hexadecimal (0-9, a-f). It returns ErrInvalidSHA256
// (wrapped with the offending value's length context) otherwise.
func ValidateSHA256(s string) error {
	if len(s) != 64 {
		return fmt.Errorf("%w: got length %d", ErrInvalidSHA256, len(s))
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		if !isDigit && !isLowerHex {
			return fmt.Errorf("%w: non-hex/uppercase char %q at index %d", ErrInvalidSHA256, string(c), i)
		}
	}
	return nil
}

// ValidateArtifactMeta validates the integrity + targeting metadata of an
// artifact: a valid SHA-256, a valid optional SHA-512, a positive size, a valid
// OS type, and non-empty board + version. It returns a descriptive, wrapped
// error on the first failure.
func ValidateArtifactMeta(m ArtifactMeta) error {
	if err := ValidateSHA256(m.SHA256); err != nil {
		return fmt.Errorf("artifact sha256: %w", err)
	}
	if m.SHA512 != "" {
		if err := validateSHA512(m.SHA512); err != nil {
			return fmt.Errorf("artifact sha512: %w", err)
		}
	}
	if m.Size <= 0 {
		return fmt.Errorf("%w: size must be > 0, got %d", ErrInvalidValue, m.Size)
	}
	if m.Size > math.MaxInt32 {
		return fmt.Errorf("%w: size %d exceeds maximum allowed", ErrInvalidValue, m.Size)
	}
	if !m.OSType.Valid() {
		return fmt.Errorf("artifact os_type: %w: %q", ErrInvalidEnum, m.OSType.String())
	}
	if m.Board == "" {
		return fmt.Errorf("%w: board", ErrMissingField)
	}
	if m.Version == "" {
		return fmt.Errorf("%w: version", ErrMissingField)
	}
	if len(m.Version) > 255 {
		return fmt.Errorf("%w: version too long (%d chars)", ErrInvalidValue, len(m.Version))
	}
	if m.Signature == "" {
		return fmt.Errorf("%w: signature", ErrMissingField)
	}
	return nil
}

// ValidateUpdateCheckRequest validates an update-check request. CurrentVersion
// is optional, but when present it must be a non-blank version string.
func ValidateUpdateCheckRequest(r UpdateCheckRequest) error {
	// CurrentVersion is optional; nothing is required. A present-but-blank value
	// (e.g. all spaces) is treated as invalid.
	if r.CurrentVersion != "" && isBlank(r.CurrentVersion) {
		return fmt.Errorf("%w: current_version is blank", ErrInvalidValue)
	}
	return nil
}

// ValidateTelemetryReport validates a telemetry report: required device_id and
// deployment_id, a valid telemetry event, a 0–100 progress, and a non-zero
// timestamp. error_code is optional.
func ValidateTelemetryReport(r TelemetryReport) error {
	if r.DeviceID == "" {
		return fmt.Errorf("%w: device_id", ErrMissingField)
	}
	if r.DeploymentID == "" {
		return fmt.Errorf("%w: deployment_id", ErrMissingField)
	}
	if !r.Event.Valid() {
		return fmt.Errorf("telemetry event: %w: %q", ErrInvalidEnum, r.Event.String())
	}
	if r.Progress < 0 || r.Progress > 100 {
		return fmt.Errorf("%w: progress must be 0..100, got %d", ErrInvalidValue, r.Progress)
	}
	if r.Timestamp.IsZero() {
		return fmt.Errorf("%w: timestamp", ErrMissingField)
	}
	// Optional telemetry annotations: when present they must be non-negative
	// (a negative duration or byte count is malformed, never silently accepted —
	// §11.4.6). Absent (nil) is the byte-identical legacy case and stays valid.
	if r.DurationMS != nil && *r.DurationMS < 0 {
		return fmt.Errorf("%w: duration_ms must be >= 0, got %d", ErrInvalidValue, *r.DurationMS)
	}
	if r.BytesTransferred != nil && *r.BytesTransferred < 0 {
		return fmt.Errorf("%w: bytes_transferred must be >= 0, got %d", ErrInvalidValue, *r.BytesTransferred)
	}
	return nil
}

// validateSHA512 checks for a 128-char lowercase hex digest.
func validateSHA512(s string) error {
	if len(s) != 128 {
		return fmt.Errorf("%w: sha512 want length 128, got %d", ErrInvalidValue, len(s))
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		if !isDigit && !isLowerHex {
			return fmt.Errorf("%w: sha512 non-hex/uppercase char %q at index %d", ErrInvalidValue, string(c), i)
		}
	}
	return nil
}

// isBlank reports whether s contains only ASCII space/tab characters.
func isBlank(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return false
		}
	}
	return true
}
