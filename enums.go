package otaprotocol

import (
	"encoding/json"
	"fmt"
)

// OSType enumerates the operating-system families a device/artifact targets.
//
// Mirrors the database CHECK on devices.os_type / artifacts.os_type
// ('android','linux','windows','other'). MVP Phase-1 only uses "android"; the
// other values exist so the OS-adapter seam is contract-ready.
type OSType string

const (
	OSAndroid OSType = "android"
	OSLinux   OSType = "linux"
	OSWindows OSType = "windows"
	OSOther   OSType = "other"
)

var osTypeValues = []OSType{OSAndroid, OSLinux, OSWindows, OSOther}

// String returns the lowercase token for the OS type.
func (o OSType) String() string { return string(o) }

// Valid reports whether o is one of the enumerated OS types.
func (o OSType) Valid() bool { return enumValid(o, osTypeValues) }

// MarshalJSON encodes the OS type as its string token.
func (o OSType) MarshalJSON() ([]byte, error) { return marshalEnum(o, o.Valid, "OSType") }

// UnmarshalJSON decodes and validates an OS type string token.
func (o *OSType) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, o, "OSType") }

// ArtifactType enumerates the OTA artifact kinds.
//
// Mirrors the artifacts.artifact_type CHECK ('full','incremental','delta').
// MVP only uses "full"; the others are reserved for delta updates (deferred).
type ArtifactType string

const (
	ArtifactFull        ArtifactType = "full"
	ArtifactIncremental ArtifactType = "incremental"
	ArtifactDelta       ArtifactType = "delta"
)

var artifactTypeValues = []ArtifactType{ArtifactFull, ArtifactIncremental, ArtifactDelta}

// String returns the lowercase token for the artifact type.
func (a ArtifactType) String() string { return string(a) }

// Valid reports whether a is one of the enumerated artifact types.
func (a ArtifactType) Valid() bool { return enumValid(a, artifactTypeValues) }

// MarshalJSON encodes the artifact type as its string token.
func (a ArtifactType) MarshalJSON() ([]byte, error) { return marshalEnum(a, a.Valid, "ArtifactType") }

// UnmarshalJSON decodes and validates an artifact type string token.
func (a *ArtifactType) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, a, "ArtifactType") }

// UploadStatus enumerates the server-side artifact validation outcome.
//
// The database models a richer set ('pending','validating','validated',
// 'failed'); the protocol surface exposes the three terminal-relevant states
// required by the contract: pending, validated, failed.
type UploadStatus string

const (
	UploadPending   UploadStatus = "pending"
	UploadValidated UploadStatus = "validated"
	UploadFailed    UploadStatus = "failed"
)

var uploadStatusValues = []UploadStatus{UploadPending, UploadValidated, UploadFailed}

// String returns the lowercase token for the upload status.
func (u UploadStatus) String() string { return string(u) }

// Valid reports whether u is one of the enumerated upload statuses.
func (u UploadStatus) Valid() bool { return enumValid(u, uploadStatusValues) }

// MarshalJSON encodes the upload status as its string token.
func (u UploadStatus) MarshalJSON() ([]byte, error) { return marshalEnum(u, u.Valid, "UploadStatus") }

// UnmarshalJSON decodes and validates an upload status string token.
func (u *UploadStatus) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, u, "UploadStatus") }

// ReleaseChannel enumerates the publish channels a release belongs to.
//
// Mirrors the releases.channel CHECK ('stable','beta','canary','internal').
type ReleaseChannel string

const (
	ChannelStable   ReleaseChannel = "stable"
	ChannelBeta     ReleaseChannel = "beta"
	ChannelCanary   ReleaseChannel = "canary"
	ChannelInternal ReleaseChannel = "internal"
)

var releaseChannelValues = []ReleaseChannel{ChannelStable, ChannelBeta, ChannelCanary, ChannelInternal}

// String returns the lowercase token for the release channel.
func (c ReleaseChannel) String() string { return string(c) }

// Valid reports whether c is one of the enumerated release channels.
func (c ReleaseChannel) Valid() bool { return enumValid(c, releaseChannelValues) }

// MarshalJSON encodes the release channel as its string token.
func (c ReleaseChannel) MarshalJSON() ([]byte, error) {
	return marshalEnum(c, c.Valid, "ReleaseChannel")
}

// UnmarshalJSON decodes and validates a release channel string token.
func (c *ReleaseChannel) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, c, "ReleaseChannel") }

// DeploymentStatus enumerates the lifecycle of a deployment (the control-plane
// instruction to deliver a release to a target set).
//
// Mirrors the deployments.status CHECK
// ('draft','active','paused','completed','failed','cancelled').
type DeploymentStatus string

const (
	DeploymentDraft     DeploymentStatus = "draft"
	DeploymentActive    DeploymentStatus = "active"
	DeploymentPaused    DeploymentStatus = "paused"
	DeploymentCompleted DeploymentStatus = "completed"
	DeploymentFailed    DeploymentStatus = "failed"
	DeploymentCancelled DeploymentStatus = "cancelled"
)

var deploymentStatusValues = []DeploymentStatus{
	DeploymentDraft, DeploymentActive, DeploymentPaused,
	DeploymentCompleted, DeploymentFailed, DeploymentCancelled,
}

// String returns the lowercase token for the deployment status.
func (d DeploymentStatus) String() string { return string(d) }

// Valid reports whether d is one of the enumerated deployment statuses.
func (d DeploymentStatus) Valid() bool { return enumValid(d, deploymentStatusValues) }

// MarshalJSON encodes the deployment status as its string token.
func (d DeploymentStatus) MarshalJSON() ([]byte, error) {
	return marshalEnum(d, d.Valid, "DeploymentStatus")
}

// UnmarshalJSON decodes and validates a deployment status string token.
func (d *DeploymentStatus) UnmarshalJSON(b []byte) error {
	return unmarshalEnum(b, d, "DeploymentStatus")
}

// DeviceDeploymentStatus enumerates the per-device delivery lifecycle.
//
// Mirrors device_deployments.status
// ('pending','downloading','installing','verifying','success','failed',
// 'rolled_back'), where rolled_back records the automatic A/B boot-failure
// rollback that is in scope for MVP.
type DeviceDeploymentStatus string

const (
	DeviceDeployPending     DeviceDeploymentStatus = "pending"
	DeviceDeployDownloading DeviceDeploymentStatus = "downloading"
	DeviceDeployInstalling  DeviceDeploymentStatus = "installing"
	DeviceDeployVerifying   DeviceDeploymentStatus = "verifying"
	DeviceDeploySuccess     DeviceDeploymentStatus = "success"
	DeviceDeployFailed      DeviceDeploymentStatus = "failed"
	DeviceDeployRolledBack  DeviceDeploymentStatus = "rolled_back"
)

var deviceDeploymentStatusValues = []DeviceDeploymentStatus{
	DeviceDeployPending, DeviceDeployDownloading, DeviceDeployInstalling,
	DeviceDeployVerifying, DeviceDeploySuccess, DeviceDeployFailed, DeviceDeployRolledBack,
}

// String returns the lowercase token for the device deployment status.
func (d DeviceDeploymentStatus) String() string { return string(d) }

// Valid reports whether d is one of the enumerated device deployment statuses.
func (d DeviceDeploymentStatus) Valid() bool { return enumValid(d, deviceDeploymentStatusValues) }

// MarshalJSON encodes the device deployment status as its string token.
func (d DeviceDeploymentStatus) MarshalJSON() ([]byte, error) {
	return marshalEnum(d, d.Valid, "DeviceDeploymentStatus")
}

// UnmarshalJSON decodes and validates a device deployment status string token.
func (d *DeviceDeploymentStatus) UnmarshalJSON(b []byte) error {
	return unmarshalEnum(b, d, "DeviceDeploymentStatus")
}

// TelemetryEvent enumerates the device-reported lifecycle events.
//
// Per master design §9 and the OpenAPI TelemetryEventType this is EXACTLY the
// six values below. It is intentionally narrower than the device UpdateState
// (which also includes "idle") because an idle device emits no telemetry event.
type TelemetryEvent string

const (
	EventDownloadStarted TelemetryEvent = "download_started"
	EventInstalling      TelemetryEvent = "installing"
	EventInstalled       TelemetryEvent = "installed"
	EventVerifying       TelemetryEvent = "verifying"
	EventSuccess         TelemetryEvent = "success"
	EventFailure         TelemetryEvent = "failure"
)

var telemetryEventValues = []TelemetryEvent{
	EventDownloadStarted, EventInstalling, EventInstalled,
	EventVerifying, EventSuccess, EventFailure,
}

// String returns the lowercase token for the telemetry event.
func (e TelemetryEvent) String() string { return string(e) }

// Valid reports whether e is one of the exactly six telemetry events.
func (e TelemetryEvent) Valid() bool { return enumValid(e, telemetryEventValues) }

// MarshalJSON encodes the telemetry event as its string token.
func (e TelemetryEvent) MarshalJSON() ([]byte, error) {
	return marshalEnum(e, e.Valid, "TelemetryEvent")
}

// UnmarshalJSON decodes and validates a telemetry event string token.
func (e *TelemetryEvent) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, e, "TelemetryEvent") }

// TargetType enumerates how a deployment selects its target set.
//
// Mirrors deployments.target_type CHECK ('all','group','device').
type TargetType string

const (
	TargetAll    TargetType = "all"
	TargetGroup  TargetType = "group"
	TargetDevice TargetType = "device"
)

var targetTypeValues = []TargetType{TargetAll, TargetGroup, TargetDevice}

// String returns the lowercase token for the target type.
func (t TargetType) String() string { return string(t) }

// Valid reports whether t is one of the enumerated target types.
func (t TargetType) Valid() bool { return enumValid(t, targetTypeValues) }

// MarshalJSON encodes the target type as its string token.
func (t TargetType) MarshalJSON() ([]byte, error) { return marshalEnum(t, t.Valid, "TargetType") }

// UnmarshalJSON decodes and validates a target type string token.
func (t *TargetType) UnmarshalJSON(b []byte) error { return unmarshalEnum(b, t, "TargetType") }

// --- generic enum helpers -------------------------------------------------

// enumStr constrains the generic helpers to string-backed enum types.
type enumStr interface{ ~string }

// enumValid reports whether v appears in the allowed set.
func enumValid[T comparable](v T, allowed []T) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

// marshalEnum encodes a string-backed enum as a JSON string, refusing to
// serialize an invalid value (so bad data cannot silently round-trip out).
func marshalEnum[T enumStr](v T, valid func() bool, typeName string) ([]byte, error) {
	if !valid() {
		return nil, fmt.Errorf("%w: %s %q", ErrInvalidEnum, typeName, string(v))
	}
	return json.Marshal(string(v))
}

// unmarshalEnum decodes a JSON string into a string-backed enum and validates
// it against the type's allowed set, returning ErrInvalidEnum on an unknown
// token.
func unmarshalEnum[T enumStr](b []byte, dst *T, typeName string) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("%s: %w", typeName, err)
	}
	v := T(s)
	if !isValidEnum(v) {
		return fmt.Errorf("%w: %s %q", ErrInvalidEnum, typeName, s)
	}
	*dst = v
	return nil
}

// validator is implemented by every enum type in this package.
type validator interface{ Valid() bool }

// isValidEnum runs the type's own Valid method via the validator interface.
func isValidEnum[T enumStr](v T) bool {
	if vv, ok := any(v).(validator); ok {
		return vv.Valid()
	}
	return false
}
