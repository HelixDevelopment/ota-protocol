// Package otaprotocol defines the shared wire protocol and contracts for the
// Helix OTA control plane (1.0.0-MVP).
//
// It is the decoupled keystone that other Helix OTA modules import: it contains
// ONLY pure contracts — typed enums, the request/response structs that match the
// REST OpenAPI schemas, their JSON (de)serialization, and validation helpers.
//
// The package deliberately contains NO I/O, NO HTTP, NO database, and NO
// transport code. It depends on the Go standard library only, so it builds
// anywhere and can be safely vendored by every Helix OTA submodule.
//
// The contracts here mirror the committed specifications:
//
//   - api/openapi.yaml and api/endpoints.md — the REST /api/v1 surface
//     (artifact intake, releases, deployments, device registration, the device
//     update-check + telemetry endpoints).
//   - database/schema.md — the helix_ota relational model whose enumerated
//     domains (os_type, artifact_type, upload_status, channel, statuses,
//     telemetry event types, target types) are reproduced as Go enums.
//
// # Enums
//
// Every enum implements [fmt.Stringer], a Valid method, and marshals to / from
// JSON as its lowercase string token. Unknown tokens are rejected on unmarshal.
//
// # Update-check "no update" (204) semantics
//
// The device update-check endpoint (GET /api/v1/client/update) returns either
// 200 with an [UpdateAvailable] body or 204 No Content when the device is
// already on target. Because 204 carries no body, the "no update" outcome is
// modeled explicitly by [UpdateCheckResult] rather than by a nil pointer, so
// callers can branch on intent without guessing.
//
// # PayloadProperties
//
// [PayloadProperties] models the AOSP update_engine header key/values
// (FILE_HASH, FILE_SIZE, METADATA_HASH, METADATA_SIZE) that are passed straight
// through to applyPayload. It can render itself to the header map and parse a
// header map back into a typed struct.
package otaprotocol
