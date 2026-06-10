package otaprotocol

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

var fixedTime = time.Date(2026, 6, 7, 0, 14, 0, 0, time.UTC)

const validSHA = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

// roundTrip marshals v to JSON and unmarshals into a fresh value of the same
// type, returning the decoded value for comparison.
func roundTrip[T any](t *testing.T, v T) T {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", b, err)
	}
	return out
}

func TestArtifactMetaRoundTrip(t *testing.T) {
	in := ArtifactMeta{
		SHA256:    validSHA,
		SHA512:    "",
		Size:      379074366,
		OSType:    OSAndroid,
		Board:     "OrangePi5Max",
		Version:   "1.1.0",
		Signature: "BASE64SIG",
		PayloadProperties: map[string]string{
			HeaderFileHash: "abc", HeaderFileSize: "10",
			HeaderMetadataHash: "def", HeaderMetadataSize: "20",
		},
	}
	if got := roundTrip(t, in); !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
}

func TestArtifactRoundTrip(t *testing.T) {
	in := Artifact{
		ArtifactID: "a91c-uuid",
		SHA256:     validSHA,
		Size:       123,
		OSType:     OSAndroid,
		Board:      "OrangePi5Max",
		Version:    "1.1.0",
		Type:       ArtifactFull,
		Status:     UploadValidated,
		StorageRef: "s3://helix-artifacts/a91c",
		Verified:   true,
		UploadedAt: fixedTime,
	}
	if got := roundTrip(t, in); !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
}

func TestReleaseRoundTrip(t *testing.T) {
	in := Release{
		ReleaseID:         "r77e-uuid",
		ArtifactID:        "a91c-uuid",
		Version:           "1.1.0",
		OSType:            OSAndroid,
		Board:             "OrangePi5Max",
		Channel:           ChannelStable,
		Status:            "published",
		Notes:             "Security patch",
		MinCurrentVersion: "1.0.0",
		CreatedAt:         fixedTime,
	}
	if got := roundTrip(t, in); !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
}

func TestDeviceRegistrationRequestRoundTrip(t *testing.T) {
	in := DeviceRegistrationRequest{
		HardwareID:     "rk3588-AABBCCDD",
		Model:          "OrangePi5Max",
		OSType:         OSAndroid,
		OSVersion:      "15",
		CurrentVersion: "1.0.0",
		Group:          "field-fleet-a",
		Metadata:       map[string]string{"region": "eu-west"},
	}
	if got := roundTrip(t, in); !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
}

func TestDeviceRegistrationResponseRoundTrip(t *testing.T) {
	in := DeviceRegistrationResponse{
		DeviceID:     "8f3a-uuid",
		HardwareID:   "rk3588-AABBCCDD",
		DeviceToken:  "jwt",
		TokenType:    "Bearer",
		ExpiresIn:    86400,
		RegisteredAt: fixedTime,
	}
	if got := roundTrip(t, in); !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
}

func TestUpdateCheckRequestRoundTrip(t *testing.T) {
	in := UpdateCheckRequest{CurrentVersion: "1.0.0"}
	if got := roundTrip(t, in); !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
	// Empty optional field omits cleanly.
	b, _ := json.Marshal(UpdateCheckRequest{})
	if string(b) != "{}" {
		t.Errorf("empty UpdateCheckRequest = %s, want {}", b)
	}
}

func TestUpdateAvailableRoundTrip(t *testing.T) {
	in := UpdateAvailable{
		ReleaseID: "r77e-uuid",
		Version:   "1.1.0",
		URL:       "https://artifacts.helix.example/a91c.zip",
		Offset:    1234,
		Size:      379074366,
		SHA256:    validSHA,
		Signature: "BASE64SIG",
		PayloadProperties: map[string]string{
			HeaderFileHash: "abc", HeaderFileSize: "379074366",
			HeaderMetadataHash: "def", HeaderMetadataSize: "46866",
		},
	}
	if got := roundTrip(t, in); !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
}

// TestUpdateAvailableDeploymentID proves the update offer carries the
// deployment_id so a device can self-serve the telemetry deployment_id from its
// own update offer (closes the §11.4.6 protocol gap surfaced by the emulator):
// the field round-trips, serialises as `deployment_id`, and is omitted when
// empty (omitempty) so legacy offers stay byte-identical.
func TestUpdateAvailableDeploymentID(t *testing.T) {
	in := UpdateAvailable{
		ReleaseID:         "r1",
		Version:           "1.1.0",
		URL:               "u",
		SHA256:            validSHA,
		Signature:         "s",
		DeploymentID:      "dep-abc-123",
		PayloadProperties: map[string]string{},
	}
	got := roundTrip(t, in)
	if got.DeploymentID != "dep-abc-123" {
		t.Errorf("DeploymentID after round-trip = %q, want %q", got.DeploymentID, "dep-abc-123")
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(b), `"deployment_id":"dep-abc-123"`) {
		t.Errorf("JSON %s missing deployment_id field", b)
	}

	// omitempty: an offer without a deployment_id must not emit the key.
	empty := UpdateAvailable{ReleaseID: "r1", Version: "1.1.0", URL: "u",
		SHA256: validSHA, Signature: "s", PayloadProperties: map[string]string{}}
	be, err := json.Marshal(empty)
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}
	if contains(string(be), "deployment_id") {
		t.Errorf("empty offer JSON %s should omit deployment_id", be)
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}

func TestUpdateCheckResultSemantics(t *testing.T) {
	// 204 no-update outcome.
	no := NoUpdate()
	if no.UpdateAvailable || no.Update != nil {
		t.Errorf("NoUpdate() = %+v, want available=false update=nil", no)
	}
	b, err := json.Marshal(no)
	if err != nil {
		t.Fatalf("marshal NoUpdate: %v", err)
	}
	if string(b) != `{"update_available":false}` {
		t.Errorf("NoUpdate JSON = %s", b)
	}

	// 200 has-update outcome.
	ua := UpdateAvailable{ReleaseID: "r1", Version: "1.1.0", URL: "u", SHA256: validSHA, Signature: "s",
		PayloadProperties: map[string]string{}}
	yes := HasUpdate(ua)
	if !yes.UpdateAvailable || yes.Update == nil {
		t.Fatalf("HasUpdate() = %+v, want available=true non-nil update", yes)
	}
	if yes.Update.ReleaseID != "r1" {
		t.Errorf("update release id = %q", yes.Update.ReleaseID)
	}

	got := roundTrip(t, yes)
	if !got.UpdateAvailable || got.Update == nil || got.Update.ReleaseID != "r1" {
		t.Errorf("round-trip has-update mismatch: %+v", got)
	}
}

func TestTelemetryReportRoundTrip(t *testing.T) {
	in := TelemetryReport{
		DeviceID:     "8f3a-uuid",
		DeploymentID: "d12b-uuid",
		Event:        EventFailure,
		Progress:     42,
		ErrorCode:    "PAYLOAD_VERIFICATION_FAILED",
		Timestamp:    fixedTime,
	}
	if got := roundTrip(t, in); !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
}

// TestTelemetryReportOptionalTelemetryFields proves the additive optional
// per-event annotations (duration_ms, bytes_transferred) round-trip with their
// exact values, serialise under the documented snake_case keys, and are omitted
// when nil so legacy reports stay byte-identical (additive, no wire break).
func TestTelemetryReportOptionalTelemetryFields(t *testing.T) {
	dur := int64(8421)
	bytes := int64(379074366)
	in := TelemetryReport{
		DeviceID:         "8f3a-uuid",
		DeploymentID:     "d12b-uuid",
		Event:            EventSuccess,
		Progress:         100,
		Timestamp:        fixedTime,
		DurationMS:       &dur,
		BytesTransferred: &bytes,
	}
	got := roundTrip(t, in)
	if got.DurationMS == nil || *got.DurationMS != 8421 {
		t.Errorf("DurationMS after round-trip = %v, want 8421", got.DurationMS)
	}
	if got.BytesTransferred == nil || *got.BytesTransferred != 379074366 {
		t.Errorf("BytesTransferred after round-trip = %v, want 379074366", got.BytesTransferred)
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(b), `"duration_ms":8421`) || !contains(string(b), `"bytes_transferred":379074366`) {
		t.Errorf("JSON %s missing duration_ms/bytes_transferred", b)
	}

	// omitempty: a report without the annotations must not emit either key, so
	// existing device payloads serialise byte-for-byte as before.
	legacy := TelemetryReport{DeviceID: "d", DeploymentID: "dep", Event: EventInstalling,
		Progress: 50, Timestamp: fixedTime}
	bl, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy: %v", err)
	}
	if contains(string(bl), "duration_ms") || contains(string(bl), "bytes_transferred") {
		t.Errorf("legacy report JSON %s should omit duration_ms/bytes_transferred", bl)
	}

	// A nil-vs-zero distinction is preserved: an explicit zero is reported, not omitted.
	zero := int64(0)
	z := TelemetryReport{DeviceID: "d", DeploymentID: "dep", Event: EventSuccess,
		Progress: 100, Timestamp: fixedTime, DurationMS: &zero, BytesTransferred: &zero}
	bz, _ := json.Marshal(z)
	if !contains(string(bz), `"duration_ms":0`) || !contains(string(bz), `"bytes_transferred":0`) {
		t.Errorf("explicit-zero report JSON %s should carry duration_ms:0 + bytes_transferred:0", bz)
	}
}

// TestTelemetryReportInvalidEnumUnmarshal confirms struct-level unmarshal fails
// when an embedded enum value is invalid.
func TestTelemetryReportInvalidEnumUnmarshal(t *testing.T) {
	js := `{"device_id":"d","deployment_id":"dep","event":"idle","progress":0,"timestamp":"2026-06-07T00:00:00Z"}`
	var r TelemetryReport
	if err := json.Unmarshal([]byte(js), &r); err == nil {
		t.Fatal("expected error unmarshalling invalid event into TelemetryReport")
	}
}
