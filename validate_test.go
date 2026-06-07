package otaprotocol

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateSHA256(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr error
	}{
		{"valid", validSHA, nil},
		{"all digits", strings.Repeat("0", 64), nil},
		{"all af", strings.Repeat("a", 64), nil},
		{"too short", strings.Repeat("a", 63), ErrInvalidSHA256},
		{"too long", strings.Repeat("a", 65), ErrInvalidSHA256},
		{"empty", "", ErrInvalidSHA256},
		{"uppercase", strings.Repeat("A", 64), ErrInvalidSHA256},
		{"non-hex g", strings.Repeat("g", 64), ErrInvalidSHA256},
		{"mixed bad char", "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a0!", ErrInvalidSHA256},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSHA256(tt.in)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("ValidateSHA256(%q) = %v, want nil", tt.in, err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateSHA256(%q) = %v, want %v", tt.in, err, tt.wantErr)
			}
		})
	}
}

func baseArtifactMeta() ArtifactMeta {
	return ArtifactMeta{
		SHA256:    validSHA,
		Size:      100,
		OSType:    OSAndroid,
		Board:     "OrangePi5Max",
		Version:   "1.1.0",
		Signature: "BASE64SIG",
	}
}

func TestValidateArtifactMeta(t *testing.T) {
	validSHA512 := strings.Repeat("a", 128)

	tests := []struct {
		name    string
		mutate  func(*ArtifactMeta)
		wantErr error
	}{
		{"valid", func(m *ArtifactMeta) {}, nil},
		{"valid with sha512", func(m *ArtifactMeta) { m.SHA512 = validSHA512 }, nil},
		{"bad sha256", func(m *ArtifactMeta) { m.SHA256 = "short" }, ErrInvalidSHA256},
		{"bad sha512 len", func(m *ArtifactMeta) { m.SHA512 = "abc" }, ErrInvalidValue},
		{"bad sha512 char", func(m *ArtifactMeta) { m.SHA512 = strings.Repeat("Z", 128) }, ErrInvalidValue},
		{"zero size", func(m *ArtifactMeta) { m.Size = 0 }, ErrInvalidValue},
		{"negative size", func(m *ArtifactMeta) { m.Size = -5 }, ErrInvalidValue},
		{"bad os", func(m *ArtifactMeta) { m.OSType = "macos" }, ErrInvalidEnum},
		{"empty board", func(m *ArtifactMeta) { m.Board = "" }, ErrMissingField},
		{"empty version", func(m *ArtifactMeta) { m.Version = "" }, ErrMissingField},
		{"empty signature", func(m *ArtifactMeta) { m.Signature = "" }, ErrMissingField},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := baseArtifactMeta()
			tt.mutate(&m)
			err := ValidateArtifactMeta(m)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("got %v, want nil", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUpdateCheckRequest(t *testing.T) {
	tests := []struct {
		name    string
		in      UpdateCheckRequest
		wantErr error
	}{
		{"empty optional ok", UpdateCheckRequest{}, nil},
		{"present version ok", UpdateCheckRequest{CurrentVersion: "1.2.3"}, nil},
		{"blank spaces invalid", UpdateCheckRequest{CurrentVersion: "   "}, ErrInvalidValue},
		{"tab invalid", UpdateCheckRequest{CurrentVersion: "\t"}, ErrInvalidValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateCheckRequest(tt.in)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("got %v, want nil", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func baseTelemetryReport() TelemetryReport {
	return TelemetryReport{
		DeviceID:     "8f3a-uuid",
		DeploymentID: "d12b-uuid",
		Event:        EventSuccess,
		Progress:     100,
		Timestamp:    fixedTime,
	}
}

func TestValidateTelemetryReport(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*TelemetryReport)
		wantErr error
	}{
		{"valid", func(r *TelemetryReport) {}, nil},
		{"valid with error_code", func(r *TelemetryReport) { r.Event = EventFailure; r.ErrorCode = "X" }, nil},
		{"progress zero ok", func(r *TelemetryReport) { r.Progress = 0 }, nil},
		{"missing device", func(r *TelemetryReport) { r.DeviceID = "" }, ErrMissingField},
		{"missing deployment", func(r *TelemetryReport) { r.DeploymentID = "" }, ErrMissingField},
		{"bad event", func(r *TelemetryReport) { r.Event = "idle" }, ErrInvalidEnum},
		{"empty event", func(r *TelemetryReport) { r.Event = "" }, ErrInvalidEnum},
		{"progress negative", func(r *TelemetryReport) { r.Progress = -1 }, ErrInvalidValue},
		{"progress over 100", func(r *TelemetryReport) { r.Progress = 101 }, ErrInvalidValue},
		{"zero timestamp", func(r *TelemetryReport) { r.Timestamp = zeroTime() }, ErrMissingField},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := baseTelemetryReport()
			tt.mutate(&r)
			err := ValidateTelemetryReport(r)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("got %v, want nil", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidationErrorsAreDescriptive ensures messages carry context, not bare
// sentinels.
func TestValidationErrorsAreDescriptive(t *testing.T) {
	err := ValidateSHA256("xyz")
	if !strings.Contains(err.Error(), "length") {
		t.Errorf("sha256 error not descriptive: %v", err)
	}
	r := baseTelemetryReport()
	r.Progress = 250
	err = ValidateTelemetryReport(r)
	if !strings.Contains(err.Error(), "250") {
		t.Errorf("telemetry progress error not descriptive: %v", err)
	}
}
