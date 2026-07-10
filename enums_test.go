package otaprotocol

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestEnumValidAndString(t *testing.T) {
	tests := []struct {
		name  string
		v     interface{ Valid() bool }
		str   string
		valid bool
	}{
		{"os android", OSAndroid, "android", true},
		{"os linux", OSLinux, "linux", true},
		{"os windows", OSWindows, "windows", true},
		{"os other", OSOther, "other", true},
		{"os bad", OSType("macos"), "macos", false},
		{"os empty", OSType(""), "", false},

		{"artifact full", ArtifactFull, "full", true},
		{"artifact incremental", ArtifactIncremental, "incremental", true},
		{"artifact delta", ArtifactDelta, "delta", true},
		{"artifact bad", ArtifactType("patch"), "patch", false},

		{"upload pending", UploadPending, "pending", true},
		{"upload validated", UploadValidated, "validated", true},
		{"upload failed", UploadFailed, "failed", true},
		{"upload bad", UploadStatus("validating"), "validating", false},

		{"channel stable", ChannelStable, "stable", true},
		{"channel beta", ChannelBeta, "beta", true},
		{"channel canary", ChannelCanary, "canary", true},
		{"channel internal", ChannelInternal, "internal", true},
		{"channel bad", ReleaseChannel("nightly"), "nightly", false},

		{"deploy draft", DeploymentDraft, "draft", true},
		{"deploy active", DeploymentActive, "active", true},
		{"deploy paused", DeploymentPaused, "paused", true},
		{"deploy completed", DeploymentCompleted, "completed", true},
		{"deploy failed", DeploymentFailed, "failed", true},
		{"deploy cancelled", DeploymentCancelled, "cancelled", true},
		{"deploy bad", DeploymentStatus("halted"), "halted", false},

		{"devdeploy pending", DeviceDeployPending, "pending", true},
		{"devdeploy downloading", DeviceDeployDownloading, "downloading", true},
		{"devdeploy installing", DeviceDeployInstalling, "installing", true},
		{"devdeploy verifying", DeviceDeployVerifying, "verifying", true},
		{"devdeploy success", DeviceDeploySuccess, "success", true},
		{"devdeploy failed", DeviceDeployFailed, "failed", true},
		{"devdeploy rolled_back", DeviceDeployRolledBack, "rolled_back", true},
		{"devdeploy bad", DeviceDeploymentStatus("done"), "done", false},

		{"event download_started", EventDownloadStarted, "download_started", true},
		{"event installing", EventInstalling, "installing", true},
		{"event installed", EventInstalled, "installed", true},
		{"event verifying", EventVerifying, "verifying", true},
		{"event success", EventSuccess, "success", true},
		{"event failure", EventFailure, "failure", true},
		{"event idle rejected", TelemetryEvent("idle"), "idle", false},
		{"event bad", TelemetryEvent("rollback"), "rollback", false},

		{"target all", TargetAll, "all", true},
		{"target group", TargetGroup, "group", true},
		{"target device", TargetDevice, "device", true},
		{"target bad", TargetType("fleet"), "fleet", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Valid(); got != tt.valid {
				t.Errorf("Valid() = %v, want %v", got, tt.valid)
			}
			if s, ok := tt.v.(interface{ String() string }); ok {
				if s.String() != tt.str {
					t.Errorf("String() = %q, want %q", s.String(), tt.str)
				}
			}
		})
	}
}

// TestTelemetryEventExactSet asserts the telemetry event enum is EXACTLY the six
// required values and nothing more.
func TestTelemetryEventExactSet(t *testing.T) {
	want := map[string]bool{
		"download_started": true, "installing": true, "installed": true,
		"verifying": true, "success": true, "failure": true,
	}
	if len(telemetryEventValues) != len(want) {
		t.Fatalf("telemetryEventValues has %d entries, want %d", len(telemetryEventValues), len(want))
	}
	for _, e := range telemetryEventValues {
		if !want[e.String()] {
			t.Errorf("unexpected telemetry event %q", e)
		}
	}
}

func TestEnumMarshalValid(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want string
	}{
		{"os", OSAndroid, `"android"`},
		{"artifact", ArtifactFull, `"full"`},
		{"upload", UploadValidated, `"validated"`},
		{"channel", ChannelStable, `"stable"`},
		{"deploy", DeploymentActive, `"active"`},
		{"devdeploy", DeviceDeploySuccess, `"success"`},
		{"event", EventSuccess, `"success"`},
		{"target", TargetAll, `"all"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.v)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(b) != tt.want {
				t.Errorf("Marshal() = %s, want %s", b, tt.want)
			}
		})
	}
}

func TestEnumMarshalInvalidRejected(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
	}{
		{"os", OSType("solaris")},
		{"artifact", ArtifactType("x")},
		{"upload", UploadStatus("x")},
		{"channel", ReleaseChannel("x")},
		{"deploy", DeploymentStatus("x")},
		{"devdeploy", DeviceDeploymentStatus("x")},
		{"event", TelemetryEvent("idle")},
		{"target", TargetType("x")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := json.Marshal(tt.v)
			if err == nil {
				t.Fatalf("Marshal(%v) succeeded, want error", tt.v)
			}
			if !errors.Is(err, ErrInvalidEnum) {
				t.Errorf("error = %v, want ErrInvalidEnum", err)
			}
		})
	}
}

func TestEnumUnmarshalValid(t *testing.T) {
	var os OSType
	if err := json.Unmarshal([]byte(`"linux"`), &os); err != nil {
		t.Fatalf("OSType unmarshal: %v", err)
	}
	if os != OSLinux {
		t.Errorf("os = %q, want linux", os)
	}

	var ev TelemetryEvent
	if err := json.Unmarshal([]byte(`"failure"`), &ev); err != nil {
		t.Fatalf("event unmarshal: %v", err)
	}
	if ev != EventFailure {
		t.Errorf("event = %q, want failure", ev)
	}

	var ch ReleaseChannel
	if err := json.Unmarshal([]byte(`"canary"`), &ch); err != nil {
		t.Fatalf("channel unmarshal: %v", err)
	}
	if ch != ChannelCanary {
		t.Errorf("channel = %q, want canary", ch)
	}
}

func TestEnumUnmarshalInvalidRejected(t *testing.T) {
	tests := []struct {
		name string
		dst  json.Unmarshaler
		in   string
	}{
		{"os unknown", new(OSType), `"beos"`},
		{"artifact unknown", new(ArtifactType), `"snapshot"`},
		{"upload unknown", new(UploadStatus), `"queued"`},
		{"channel unknown", new(ReleaseChannel), `"edge"`},
		{"deploy unknown", new(DeploymentStatus), `"halted"`},
		{"devdeploy unknown", new(DeviceDeploymentStatus), `"done"`},
		{"event idle", new(TelemetryEvent), `"idle"`},
		{"target unknown", new(TargetType), `"region"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dst.UnmarshalJSON([]byte(tt.in))
			if err == nil {
				t.Fatalf("UnmarshalJSON(%s) succeeded, want error", tt.in)
			}
			if !errors.Is(err, ErrInvalidEnum) {
				t.Errorf("error = %v, want ErrInvalidEnum", err)
			}
		})
	}
}

func TestEnumUnmarshalNonStringRejected(t *testing.T) {
	var os OSType
	if err := json.Unmarshal([]byte(`123`), &os); err == nil {
		t.Fatal("expected error unmarshalling a number into OSType")
	}
}

// TestEnumUnmarshalNullRejected proves a JSON `null` decoded into an enum
// field is REJECTED, not silently coerced to a valid-looking zero value.
// encoding/json's documented null-into-non-pointer behavior is a no-op (the
// destination string is left at its zero value "" with a nil error) — a
// naive enum decoder could therefore let `null` slip through as an
// unvalidated empty string. unmarshalEnum does not: the zero-value "" it
// observes after the no-op inner Unmarshal is then run through isValidEnum,
// which every production enum's allowed-set rejects (none enumerates "" as
// valid), so the outer call correctly returns ErrInvalidEnum both for a
// bare `null` token and for `null` embedded inside a larger struct's JSON.
func TestEnumUnmarshalNullRejected(t *testing.T) {
	var os OSType
	err := json.Unmarshal([]byte(`null`), &os)
	if err == nil {
		t.Fatal("json null unmarshalled into OSType without error, want ErrInvalidEnum")
	}
	if !errors.Is(err, ErrInvalidEnum) {
		t.Fatalf("error = %v, want ErrInvalidEnum", err)
	}
	if os != "" {
		t.Fatalf("rejected null unmarshal mutated dst to %q, want unchanged empty", os)
	}

	js := `{"artifact_id":"a","sha256":"` + validSHA + `","size":1,"os_type":null,"board":"b","version":"1","artifact_type":"full","upload_status":"validated"}`
	var a Artifact
	if err := json.Unmarshal([]byte(js), &a); !errors.Is(err, ErrInvalidEnum) {
		t.Fatalf("struct-embedded null os_type = %v, want ErrInvalidEnum", err)
	}
}

// TestEnumRoundTrip confirms every valid enum survives marshal then unmarshal.
func TestEnumRoundTrip(t *testing.T) {
	osVals := osTypeValues
	for _, v := range osVals {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal %q: %v", v, err)
		}
		var got OSType
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		if got != v {
			t.Errorf("round-trip %q -> %q", v, got)
		}
	}
}
