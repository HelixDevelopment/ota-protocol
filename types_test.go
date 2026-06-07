package otaprotocol

import (
	"encoding/json"
	"reflect"
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

// TestTelemetryReportInvalidEnumUnmarshal confirms struct-level unmarshal fails
// when an embedded enum value is invalid.
func TestTelemetryReportInvalidEnumUnmarshal(t *testing.T) {
	js := `{"device_id":"d","deployment_id":"dep","event":"idle","progress":0,"timestamp":"2026-06-07T00:00:00Z"}`
	var r TelemetryReport
	if err := json.Unmarshal([]byte(js), &r); err == nil {
		t.Fatal("expected error unmarshalling invalid event into TelemetryReport")
	}
}
