package otaprotocol

import "time"

// ArtifactMeta is the integrity + targeting metadata declared for an OTA
// artifact. It corresponds to the upload metadata part (ArtifactUploadMetadata)
// and the integrity columns of the artifacts table.
//
// SHA256 is mandatory (lowercase hex). SHA512 is optional ("where available").
// PayloadProperties carries the AOSP update_engine header values.
type ArtifactMeta struct {
	SHA256            string            `json:"sha256"`
	SHA512            string            `json:"sha512,omitempty"`
	Size              int64             `json:"size"`
	OSType            OSType            `json:"os_type"`
	Board             string            `json:"board"`
	Version           string            `json:"version"`
	Signature         string            `json:"signature"`
	PayloadProperties map[string]string `json:"payload_properties,omitempty"`
}

// Artifact is the stored-and-verified artifact metadata returned by the
// artifact endpoints (Artifact schema).
type Artifact struct {
	ArtifactID string       `json:"artifact_id"`
	SHA256     string       `json:"sha256"`
	SHA512     string       `json:"sha512,omitempty"`
	Size       int64        `json:"size"`
	OSType     OSType       `json:"os_type"`
	Board      string       `json:"board"`
	Version    string       `json:"version"`
	Type       ArtifactType `json:"artifact_type"`
	Status     UploadStatus `json:"upload_status"`
	StorageRef string       `json:"storage_ref,omitempty"`
	Verified   bool         `json:"verified"`
	UploadedAt time.Time    `json:"uploaded_at"`
}

// Release binds a validated artifact to a published, deployable version on a
// channel (Release schema / releases table).
type Release struct {
	ReleaseID         string         `json:"release_id"`
	ArtifactID        string         `json:"artifact_id"`
	Version           string         `json:"version"`
	OSType            OSType         `json:"os_type"`
	Board             string         `json:"board"`
	Channel           ReleaseChannel `json:"channel"`
	Status            string         `json:"status"`
	Notes             string         `json:"notes,omitempty"`
	MinCurrentVersion string         `json:"min_current_version,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
}

// DeviceRegistrationRequest provisions a device in the registry
// (DeviceRegistration schema).
type DeviceRegistrationRequest struct {
	HardwareID     string            `json:"hardware_id"`
	Model          string            `json:"model"`
	OSType         OSType            `json:"os_type"`
	OSVersion      string            `json:"os_version,omitempty"`
	CurrentVersion string            `json:"current_version,omitempty"`
	Group          string            `json:"group,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// DeviceRegistrationResponse is returned after a device is registered; it mints
// the device-scoped bearer token (DeviceRegistered schema).
type DeviceRegistrationResponse struct {
	DeviceID     string    `json:"device_id"`
	HardwareID   string    `json:"hardware_id"`
	DeviceToken  string    `json:"device_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RegisteredAt time.Time `json:"registered_at"`
}

// UpdateCheckRequest carries the optional client-reported version that lets the
// server short-circuit the update check. The effective device id is the
// bearer-token subject, so it is not part of the request body.
type UpdateCheckRequest struct {
	CurrentVersion string `json:"current_version,omitempty"`
}

// UpdateAvailable is the 200-response apply instruction for
// update_engine.applyPayload(url, offset, size, headers) (UpdateAvailable
// schema). The url is Range-served, Content-Encoding identity, ZIP_STORED.
type UpdateAvailable struct {
	ReleaseID string `json:"release_id"`
	// DeploymentID is the active deployment this offer belongs to. The device
	// echoes it back in every telemetry report (POST /client/telemetry), whose
	// schema validator REQUIRES a deployment_id. Carrying it here lets a device
	// self-serve the id from its own update offer instead of obtaining it
	// out-of-band; absent (omitempty) for legacy offers, where the device falls
	// back to an operator-supplied id.
	DeploymentID      string            `json:"deployment_id,omitempty"`
	Version           string            `json:"version"`
	URL               string            `json:"url"`
	Offset            int64             `json:"offset"`
	Size              int64             `json:"size"`
	SHA256            string            `json:"sha256"`
	Signature         string            `json:"signature"`
	PayloadProperties map[string]string `json:"payload_properties"`
	// Delta, when non-nil, offers a smaller base->target delta payload the device
	// MAY apply instead of the full payload above, IFF its current version equals
	// Delta.BaseVersion. Absent when no applicable delta exists; the full payload
	// is always the safe fallback (delta_updates_design.md).
	Delta *DeltaUpdate `json:"delta,omitempty"`
}

// DeltaUpdate is the optional delta-payload offer within an UpdateAvailable. The
// device applies it only when on BaseVersion; otherwise it uses the full payload.
type DeltaUpdate struct {
	BaseVersion string `json:"base_version"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
}

// UpdateCheckResult models the two possible outcomes of GET /client/update in a
// single value so the 204 "no update" semantics are explicit rather than
// implied by a nil pointer.
//
// When UpdateAvailable is true, Update is populated (the 200 body). When it is
// false, the device is already on target and the server returns 204 No Content
// with no body.
type UpdateCheckResult struct {
	// UpdateAvailable reports whether an update is assigned (200) or the device
	// is up to date (204).
	UpdateAvailable bool `json:"update_available"`
	// Update holds the apply instructions; non-nil iff UpdateAvailable is true.
	Update *UpdateAvailable `json:"update,omitempty"`
}

// NoUpdate returns an UpdateCheckResult representing the 204 No Content outcome
// (device already on target).
func NoUpdate() UpdateCheckResult {
	return UpdateCheckResult{UpdateAvailable: false, Update: nil}
}

// HasUpdate returns an UpdateCheckResult representing the 200 outcome carrying
// the given apply instructions.
func HasUpdate(u UpdateAvailable) UpdateCheckResult {
	return UpdateCheckResult{UpdateAvailable: true, Update: &u}
}

// TelemetryReport is a single device lifecycle event report
// (POST /client/telemetry). The contract requires device_id, deployment_id,
// event, progress, error_code, and timestamp. A device may report only for its
// own deviceId (enforced server-side against the bearer-token subject).
type TelemetryReport struct {
	DeviceID     string         `json:"device_id"`
	DeploymentID string         `json:"deployment_id"`
	Event        TelemetryEvent `json:"event"`
	Progress     int            `json:"progress"`
	ErrorCode    string         `json:"error_code,omitempty"`
	Timestamp    time.Time      `json:"timestamp"`
}
