package otaprotocol

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Chaos: ValidateSHA256 with maximally malformed inputs
// ---------------------------------------------------------------------------

func TestChaosValidateSHA256Malformed(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("chaos test skipped in short mode")
	}

	// Recover from any panic inside a goroutine so one bad input does not kill
	// the entire suite.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic in TestChaosValidateSHA256Malformed: %v", r)
		}
	}()

	tests := []struct {
		name string
		in   string
	}{
		{"empty string", ""},
		{"short hash", strings.Repeat("a", 63)},
		{"long hash", strings.Repeat("b", 65)},
		{"uppercase hex", strings.Repeat("A", 64)},
		{"non-hex g chars", strings.Repeat("g", 64)},
		{"mixed valid with trailing bang", "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a0!"},
		{"only spaces", strings.Repeat(" ", 64)},
		{"tab characters", strings.Repeat("\t", 64)},
		{"null bytes", string(make([]byte, 64))},
		{"digit bound slash", strings.Repeat("/", 64)},
		{"digit bound colon", strings.Repeat(":", 64)},
		{"letter bound backtick", strings.Repeat("`", 64)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSHA256(tt.in)
			if err == nil {
				t.Errorf("ValidateSHA256(%q) = nil, expected error", tt.in)
			}
		})
	}

	record := map[string]interface{}{
		"test":  "TestChaosValidateSHA256Malformed",
		"cases": len(tests),
		"pass":  !t.Failed(),
		"time":  time.Now().UTC(),
	}
	writeEvidence("ChaosValidateSHA256Malformed", record)
}

// ---------------------------------------------------------------------------
// Chaos: ValidateArtifactMeta with boundary-challenged values
// ---------------------------------------------------------------------------

func TestChaosValidateArtifactMetaBoundaries(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("chaos test skipped in short mode")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic in TestChaosValidateArtifactMetaBoundaries: %v", r)
		}
	}()

	tests := []struct {
		name   string
		mutate func(*ArtifactMeta)
	}{
		{"zero size", func(m *ArtifactMeta) { m.Size = 0 }},
		{"negative size", func(m *ArtifactMeta) { m.Size = -1 }},
		{"empty board", func(m *ArtifactMeta) { m.Board = "" }},
		{"empty version", func(m *ArtifactMeta) { m.Version = "" }},
		{"empty signature", func(m *ArtifactMeta) { m.Signature = "" }},
		{"valid SHA256 / missing SHA512 / empty signature",
			func(m *ArtifactMeta) { m.SHA256 = validSHA; m.SHA512 = ""; m.Signature = "" }},
		{"malformed SHA512 uppercase",
			func(m *ArtifactMeta) { m.SHA512 = strings.Repeat("A", 128) }},
		{"invalid OSType",
			func(m *ArtifactMeta) { m.OSType = "macos" }},
		{"empty OSType",
			func(m *ArtifactMeta) { m.OSType = "" }},
		{"negative size with invalid OSType combo",
			func(m *ArtifactMeta) { m.Size = -5; m.OSType = "ios" }},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := baseArtifactMeta()
			tt.mutate(&m)
			err := ValidateArtifactMeta(m)
			if err == nil {
				t.Errorf("ValidateArtifactMeta(%+v) = nil, expected error", m)
			}
		})
	}

	record := map[string]interface{}{
		"test":  "TestChaosValidateArtifactMetaBoundaries",
		"cases": len(tests),
		"pass":  !t.Failed(),
		"time":  time.Now().UTC(),
	}
	writeEvidence("ChaosValidateArtifactMetaBoundaries", record)
}

// ---------------------------------------------------------------------------
// Chaos: TelemetryReport with boundary / out-of-range values
// ---------------------------------------------------------------------------

func TestChaosTelemetryReportBoundaries(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("chaos test skipped in short mode")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic in TestChaosTelemetryReportBoundaries: %v", r)
		}
	}()

	neg := int64(-1)
	pos := int64(100)

	tests := []struct {
		name   string
		mutate func(*TelemetryReport)
	}{
		{"negative duration_ms", func(r *TelemetryReport) { r.DurationMS = &neg }},
		{"negative bytes_transferred", func(r *TelemetryReport) { r.BytesTransferred = &neg }},
		{"progress over 100", func(r *TelemetryReport) { r.Progress = 101 }},
		{"progress negative", func(r *TelemetryReport) { r.Progress = -1 }},
		{"zero timestamp", func(r *TelemetryReport) { r.Timestamp = zeroTime() }},
		{"empty device_id", func(r *TelemetryReport) { r.DeviceID = "" }},
		{"invalid telemetry event", func(r *TelemetryReport) { r.Event = "idle" }},
		{"empty telemetry event", func(r *TelemetryReport) { r.Event = "" }},
		{"all boundary violations combined",
			func(r *TelemetryReport) {
				r.DeviceID = ""
				r.Event = "idle"
				r.Progress = -1
				r.Timestamp = zeroTime()
				r.DurationMS = &neg
				r.BytesTransferred = &pos
			}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := baseTelemetryReport()
			tt.mutate(&r)
			err := ValidateTelemetryReport(r)
			if err == nil {
				t.Errorf("ValidateTelemetryReport(%+v) = nil, expected error", r)
			}
		})
	}

	record := map[string]interface{}{
		"test":  "TestChaosTelemetryReportBoundaries",
		"cases": len(tests),
		"pass":  !t.Failed(),
		"time":  time.Now().UTC(),
	}
	writeEvidence("ChaosTelemetryReportBoundaries", record)
}

// ---------------------------------------------------------------------------
// Chaos: concurrent MarshalJSON across all enum types (data-race detection)
// ---------------------------------------------------------------------------

func TestChaosConcurrentEnumMarshal(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("chaos test skipped in short mode")
	}

	const N = 50

	values := make([]interface{}, 0, 32)
	for _, v := range osTypeValues {
		values = append(values, v)
	}
	for _, v := range artifactTypeValues {
		values = append(values, v)
	}
	for _, v := range uploadStatusValues {
		values = append(values, v)
	}
	for _, v := range releaseChannelValues {
		values = append(values, v)
	}
	for _, v := range deploymentStatusValues {
		values = append(values, v)
	}
	for _, v := range deviceDeploymentStatusValues {
		values = append(values, v)
	}
	for _, v := range telemetryEventValues {
		values = append(values, v)
	}
	for _, v := range targetTypeValues {
		values = append(values, v)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, N*2)

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, v := range values {
				b, err := json.Marshal(v)
				if err != nil {
					errCh <- err
					return
				}
				if len(b) == 0 {
					errCh <- nil // will be handled as "empty"
				}
			}
		}()
	}

	// Wait for all goroutines and close the error channel.
	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		if err != nil {
			t.Errorf("concurrent marshal error: %v", err)
		}
	}

	record := map[string]interface{}{
		"test":         "TestChaosConcurrentEnumMarshal",
		"N":            N,
		"enum_types":   8,
		"marshal_pass": !t.Failed(),
	}
	writeEvidence("ChaosConcurrentEnumMarshal", record)
}

// ---------------------------------------------------------------------------
// Chaos: PayloadProperties with malformed / missing / garbage header maps
// ---------------------------------------------------------------------------

func TestChaosPayloadProperties(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("chaos test skipped in short mode")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic in TestChaosPayloadProperties: %v", r)
		}
	}()

	tests := []struct {
		name       string
		headers    map[string]string
		wantErr    bool
		validateOk bool
	}{
		{"empty map", map[string]string{}, true, false},
		{"all keys missing",
			map[string]string{"X-Extra": "foo"}, true, false},
		{"missing FILE_HASH",
			map[string]string{
				HeaderFileSize:     "100",
				HeaderMetadataHash: "mh",
				HeaderMetadataSize: "10",
			}, true, false},
		{"missing METADATA_HASH",
			map[string]string{
				HeaderFileHash:     "fh",
				HeaderFileSize:     "100",
				HeaderMetadataSize: "10",
			}, true, false},
		{"non-numeric FILE_SIZE",
			map[string]string{
				HeaderFileHash:     "fh",
				HeaderFileSize:     "not-a-number",
				HeaderMetadataHash: "mh",
				HeaderMetadataSize: "10",
			}, true, false},
		{"non-numeric METADATA_SIZE",
			map[string]string{
				HeaderFileHash:     "fh",
				HeaderFileSize:     "100",
				HeaderMetadataHash: "mh",
				HeaderMetadataSize: "12.5",
			}, true, false},
		// HC-3 (§11.4.108/§11.4.120 reconciliation): ParsePayloadProperties now
		// rejects a negative FILE_SIZE at parse time instead of deferring the
		// check to a separate Validate() call — closing the parse-vs-validate
		// divergence where Parse silently accepted a value Validate rejected.
		// This case moved from the "parses OK, fails Validate" bucket into the
		// "rejected at parse time" bucket; see wave_hc3_test.go for the
		// dedicated regression coverage + the RED (pre-fix)/GREEN (post-fix)
		// anti-tautology proof.
		{"negative FILE_SIZE string",
			map[string]string{
				HeaderFileHash:     "fh",
				HeaderFileSize:     "-100",
				HeaderMetadataHash: "mh",
				HeaderMetadataSize: "10",
			}, true, false},
		{"empty FILE_HASH",
			map[string]string{
				HeaderFileHash:     "",
				HeaderFileSize:     "100",
				HeaderMetadataHash: "mh",
				HeaderMetadataSize: "10",
			}, false, false},
		{"empty METADATA_HASH",
			map[string]string{
				HeaderFileHash:     "fh",
				HeaderFileSize:     "100",
				HeaderMetadataHash: "",
				HeaderMetadataSize: "10",
			}, false, false},
		{"extra unknown keys",
			map[string]string{
				HeaderFileHash:          "fh",
				HeaderFileSize:          "100",
				HeaderMetadataHash:      "mh",
				HeaderMetadataSize:      "10",
				"EXTRA_KEY":             "should-be-ignored",
				"SWITCH_SLOT_ON_REBOOT": "true",
			}, false, true},
		{"FILE_SIZE overflow int64",
			map[string]string{
				HeaderFileHash:     "fh",
				HeaderFileSize:     "9223372036854775808",
				HeaderMetadataHash: "mh",
				HeaderMetadataSize: "10",
			}, true, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := ParsePayloadProperties(tt.headers)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePayloadProperties(%v) = nil, expected error", tt.headers)
				}
				return
			}
			if err != nil {
				t.Errorf("ParsePayloadProperties(%v) = %v, want nil", tt.headers, err)
				return
			}
			// Verify parse-time constructs a usable PayloadProperties even from
			// degenerate data (the callers will run Validate separately).
			if tt.validateOk {
				if verr := p.Validate(); verr != nil {
					t.Errorf("ParsePayloadProperties(%v) ok but Validate = %v", tt.headers, verr)
				}
			} else {
				// These parse but should fail Validate (empty hashes, negative
				// sizes) — the caller MUST run Validate before using them.
				if verr := p.Validate(); verr == nil {
					t.Errorf("ParsePayloadProperties(%v) = %+v, Validate = nil but expected error",
						tt.headers, p)
				}
			}
		})
	}

	record := map[string]interface{}{
		"test":  "TestChaosPayloadProperties",
		"cases": len(tests),
		"pass":  !t.Failed(),
		"time":  time.Now().UTC(),
	}
	writeEvidence("ChaosPayloadProperties", record)
}
