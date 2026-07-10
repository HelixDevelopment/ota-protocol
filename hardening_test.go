package otaprotocol

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
)

// nonValidatorEnum is a test-only string-backed enum type that deliberately
// does NOT implement the validator interface (no Valid() method). It exists to
// exercise the defensive fallback branch of isValidEnum: when a value passed to
// the generic helper does not satisfy validator, the helper MUST report it
// invalid (return false) rather than panicking or silently accepting it. Every
// production enum implements Valid(), so this branch is only reachable with a
// purpose-built type — its contract is "unknown/typeless => invalid", which is
// the safe, anti-bluff default (§11.4.6: never silently accept unverified
// input).
type nonValidatorEnum string

// TestIsValidEnumFallbackRejectsNonValidator proves isValidEnum returns false
// for a string-backed enum that lacks a Valid() method, and that unmarshalEnum
// — which delegates to isValidEnum — therefore REJECTS such a token with
// ErrInvalidEnum instead of accepting garbage. A regression that made the
// fallback "return true" (accept-anything) would fail here.
func TestIsValidEnumFallbackRejectsNonValidator(t *testing.T) {
	if isValidEnum(nonValidatorEnum("anything")) {
		t.Fatal("isValidEnum accepted a non-validator type; the defensive fallback must reject (return false)")
	}
	if isValidEnum(nonValidatorEnum("")) {
		t.Fatal("isValidEnum accepted an empty non-validator value; must reject")
	}

	// unmarshalEnum routes through isValidEnum: a non-validator type must be
	// rejected, never silently round-tripped in.
	var dst nonValidatorEnum
	err := unmarshalEnum([]byte(`"whatever"`), &dst, "nonValidatorEnum")
	if err == nil {
		t.Fatal("unmarshalEnum accepted a non-validator token; want ErrInvalidEnum")
	}
	if !errors.Is(err, ErrInvalidEnum) {
		t.Fatalf("unmarshalEnum error = %v, want ErrInvalidEnum", err)
	}
	// Rejection must leave the destination untouched (no partial write).
	if dst != "" {
		t.Fatalf("rejected unmarshal mutated dst to %q, want unchanged empty", dst)
	}
}

// TestPayloadPropertiesValidateMetadataSizeNegative covers the MetadataSize < 0
// branch of PayloadProperties.Validate, which the existing table test does not
// exercise (it flips FileSize but not MetadataSize). A negative metadata size is
// a malformed AOSP header that MUST be rejected, never passed through to
// update_engine.
func TestPayloadPropertiesValidateMetadataSizeNegative(t *testing.T) {
	p := PayloadProperties{
		FileHash:     "fh",
		FileSize:     10,
		MetadataHash: "mh",
		MetadataSize: -1,
	}
	err := p.Validate()
	if !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("Validate() with MetadataSize=-1 = %v, want ErrInvalidValue", err)
	}
	if !strings.Contains(err.Error(), HeaderMetadataSize) {
		t.Errorf("error %q does not name the offending header %q", err, HeaderMetadataSize)
	}
}

// TestPayloadPropertiesValidateMissingMetadataHash covers the empty
// MetadataHash branch (the existing test only flips FileHash). An empty
// integrity hash must be rejected as a missing required field.
func TestPayloadPropertiesValidateMissingMetadataHash(t *testing.T) {
	p := PayloadProperties{FileHash: "fh", FileSize: 1, MetadataHash: "", MetadataSize: 1}
	err := p.Validate()
	if !errors.Is(err, ErrMissingField) {
		t.Fatalf("Validate() with empty MetadataHash = %v, want ErrMissingField", err)
	}
	if !strings.Contains(err.Error(), HeaderMetadataHash) {
		t.Errorf("error %q does not name %q", err, HeaderMetadataHash)
	}
}

// TestPayloadPropertiesHeadersRoundTripExact asserts Headers() then
// ParsePayloadProperties is lossless for the full field set including the
// integer-string boundary (large positive, and zero). This is the wire contract
// the device's applyPayload depends on: a size that does not survive the
// string<->int64 round-trip would corrupt the update offer.
func TestPayloadPropertiesHeadersRoundTripExact(t *testing.T) {
	cases := []PayloadProperties{
		{FileHash: "a1b2", FileSize: 0, MetadataHash: "c3d4", MetadataSize: 0},
		{FileHash: "f", FileSize: 9223372036854775807, MetadataHash: "m", MetadataSize: 4096},
		{FileHash: "h", FileSize: 379074366, MetadataHash: "mh", MetadataSize: 1},
	}
	for _, want := range cases {
		got, err := ParsePayloadProperties(want.Headers())
		if err != nil {
			t.Fatalf("ParsePayloadProperties(%+v.Headers()) error: %v", want, err)
		}
		if got != want {
			t.Fatalf("round-trip mismatch: got %+v, want %+v", got, want)
		}
	}
}

// TestParsePayloadPropertiesMissingFileSize covers the parseHeaderInt
// missing-key branch reached via the FILE_SIZE path (hashes present, size key
// absent). The existing payload test does not assert this exact branch.
func TestParsePayloadPropertiesMissingFileSize(t *testing.T) {
	h := map[string]string{
		HeaderFileHash:     "fh",
		HeaderMetadataHash: "mh",
		HeaderMetadataSize: "20",
		// HeaderFileSize intentionally absent.
	}
	_, err := ParsePayloadProperties(h)
	if !errors.Is(err, ErrMissingPayloadHeader) {
		t.Fatalf("ParsePayloadProperties missing FILE_SIZE = %v, want ErrMissingPayloadHeader", err)
	}
	if !strings.Contains(err.Error(), HeaderFileSize) {
		t.Errorf("error %q does not name %q", err, HeaderFileSize)
	}
}

// TestParsePayloadPropertiesNonNumericMetadataSize covers the strconv failure
// branch of parseHeaderInt via the METADATA_SIZE path (the existing test only
// corrupts FILE_SIZE). A non-numeric size header is malformed input that must
// be rejected with ErrInvalidValue, never coerced to 0.
func TestParsePayloadPropertiesNonNumericMetadataSize(t *testing.T) {
	h := map[string]string{
		HeaderFileHash:     "fh",
		HeaderMetadataHash: "mh",
		HeaderFileSize:     "100",
		HeaderMetadataSize: "not-a-number",
	}
	_, err := ParsePayloadProperties(h)
	if !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("ParsePayloadProperties non-numeric METADATA_SIZE = %v, want ErrInvalidValue", err)
	}
	if !strings.Contains(err.Error(), HeaderMetadataSize) {
		t.Errorf("error %q does not name %q", err, HeaderMetadataSize)
	}
}

// TestValidateArtifactMetaFirstFailureWins proves the validation order is
// stable and the FIRST malformed field is reported, not a later one. This pins
// the contract: a caller fixing one error and re-submitting reaches the next
// error deterministically. An artifact that is simultaneously bad-sha AND
// zero-size must surface the sha error first (sha is checked first).
func TestValidateArtifactMetaFirstFailureWins(t *testing.T) {
	m := baseArtifactMeta()
	m.SHA256 = "bad"
	m.Size = 0 // also invalid, but checked AFTER sha
	err := ValidateArtifactMeta(m)
	if !errors.Is(err, ErrInvalidSHA256) {
		t.Fatalf("simultaneous bad-sha + zero-size surfaced %v, want ErrInvalidSHA256 (first check)", err)
	}
}

// TestValidateArtifactMetaSHA512OnlyWhenPresent proves the optional SHA512 is
// only validated when non-empty: a valid artifact with empty SHA512 passes, and
// the SHA512 check is genuinely gated (a present-but-bad SHA512 fails while an
// absent one does not). Guards against a regression that made SHA512 mandatory
// or that skipped validation of a present value.
func TestValidateArtifactMetaSHA512OnlyWhenPresent(t *testing.T) {
	m := baseArtifactMeta()
	m.SHA512 = "" // absent -> must pass
	if err := ValidateArtifactMeta(m); err != nil {
		t.Fatalf("absent SHA512 rejected: %v", err)
	}
	m.SHA512 = strings.Repeat("a", 127) // present-but-wrong-length -> must fail
	if err := ValidateArtifactMeta(m); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("present bad-length SHA512 = %v, want ErrInvalidValue", err)
	}
}

// TestValidateTelemetryReportBoundaryProgress pins the exact 0..100 boundary:
// 0 and 100 inclusive pass; 101 and -1 fail. Off-by-one in the bound check
// (e.g. < 100 instead of <= 100) would fail here.
func TestValidateTelemetryReportBoundaryProgress(t *testing.T) {
	for _, p := range []int{0, 1, 50, 99, 100} {
		r := baseTelemetryReport()
		r.Progress = p
		if err := ValidateTelemetryReport(r); err != nil {
			t.Errorf("progress %d rejected, want accepted: %v", p, err)
		}
	}
	for _, p := range []int{-1, 101, 200} {
		r := baseTelemetryReport()
		r.Progress = p
		if err := ValidateTelemetryReport(r); !errors.Is(err, ErrInvalidValue) {
			t.Errorf("progress %d = %v, want ErrInvalidValue", p, err)
		}
	}
}

// TestValidateTelemetryReportZeroDurationAndBytesAccepted pins the nil-vs-zero
// semantics at the validation layer: a present (non-nil) zero is a legitimately
// reported zero and MUST pass, distinct from a negative which must fail. This
// protects the additive-optional-field contract documented on the struct.
func TestValidateTelemetryReportZeroDurationAndBytesAccepted(t *testing.T) {
	zero := int64(0)
	r := baseTelemetryReport()
	r.DurationMS = &zero
	r.BytesTransferred = &zero
	if err := ValidateTelemetryReport(r); err != nil {
		t.Fatalf("non-nil zero duration/bytes rejected: %v", err)
	}
}

// TestEnumCrossTypeIsolation proves each enum validates ONLY its own token set:
// a token valid for one enum (e.g. "android" for OSType) is rejected by an
// unrelated enum (e.g. ArtifactType). A regression that shared an allowed-set
// across types would wrongly accept cross-type tokens.
func TestEnumCrossTypeIsolation(t *testing.T) {
	if ArtifactType(OSAndroid).Valid() {
		t.Error("ArtifactType wrongly accepts OSType token 'android'")
	}
	if OSType(ArtifactFull).Valid() {
		t.Error("OSType wrongly accepts ArtifactType token 'full'")
	}
	if UploadStatus(ChannelStable).Valid() {
		t.Error("UploadStatus wrongly accepts ReleaseChannel token 'stable'")
	}
	if ReleaseChannel(UploadPending).Valid() {
		t.Error("ReleaseChannel wrongly accepts UploadStatus token 'pending'")
	}
	// Sanity: each accepts its own.
	if !OSAndroid.Valid() || !ArtifactFull.Valid() || !UploadPending.Valid() || !ChannelStable.Valid() {
		t.Fatal("an enum rejected its own valid token")
	}
}

// TestArtifactUnmarshalRejectsInjectedInvalidEnum proves an invalid enum token
// embedded inside a larger struct's JSON is rejected during decode (the custom
// UnmarshalJSON fires through the nested field), so a malicious/garbled
// artifact payload cannot smuggle in an out-of-set os_type or status. This is
// the wire-boundary anti-bluff guard: bad data does not silently round-trip in.
func TestArtifactUnmarshalRejectsInjectedInvalidEnum(t *testing.T) {
	bad := []struct {
		name string
		js   string
	}{
		{"bad os_type", `{"artifact_id":"a","sha256":"` + validSHA + `","size":1,"os_type":"macos","board":"b","version":"1","artifact_type":"full","upload_status":"validated"}`},
		{"bad artifact_type", `{"artifact_id":"a","sha256":"` + validSHA + `","size":1,"os_type":"android","board":"b","version":"1","artifact_type":"super","upload_status":"validated"}`},
		{"bad upload_status", `{"artifact_id":"a","sha256":"` + validSHA + `","size":1,"os_type":"android","board":"b","version":"1","artifact_type":"full","upload_status":"done"}`},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			var a Artifact
			err := json.Unmarshal([]byte(tc.js), &a)
			if err == nil {
				t.Fatalf("decoded artifact with %s, want rejection", tc.name)
			}
			if !errors.Is(err, ErrInvalidEnum) {
				t.Fatalf("error = %v, want ErrInvalidEnum", err)
			}
		})
	}
}

// TestReleaseUnmarshalRejectsInvalidChannel mirrors the above for Release's
// channel enum field — an out-of-set publish channel must be rejected at decode.
func TestReleaseUnmarshalRejectsInvalidChannel(t *testing.T) {
	js := `{"release_id":"r","artifact_id":"a","version":"1","os_type":"android","board":"b","channel":"nightly","status":"published","created_at":"2026-06-07T00:00:00Z"}`
	var r Release
	err := json.Unmarshal([]byte(js), &r)
	if err == nil {
		t.Fatal("decoded Release with invalid channel 'nightly', want rejection")
	}
	if !errors.Is(err, ErrInvalidEnum) {
		t.Fatalf("error = %v, want ErrInvalidEnum", err)
	}
}

// TestMarshalEnumRefusesInvalidValue proves the marshal side is symmetric with
// unmarshal: an enum holding an out-of-set value cannot be serialized out (it
// returns ErrInvalidEnum), so corrupted in-memory state cannot leak onto the
// wire. Covers every enum type's marshal guard via the shared helper.
func TestMarshalEnumRefusesInvalidValue(t *testing.T) {
	if _, err := json.Marshal(OSType("solaris")); !errors.Is(err, ErrInvalidEnum) {
		t.Errorf("marshal invalid OSType = %v, want ErrInvalidEnum", err)
	}
	if _, err := json.Marshal(ArtifactType("patch")); !errors.Is(err, ErrInvalidEnum) {
		t.Errorf("marshal invalid ArtifactType = %v, want ErrInvalidEnum", err)
	}
	if _, err := json.Marshal(UploadStatus("queued")); !errors.Is(err, ErrInvalidEnum) {
		t.Errorf("marshal invalid UploadStatus = %v, want ErrInvalidEnum", err)
	}
	if _, err := json.Marshal(ReleaseChannel("edge")); !errors.Is(err, ErrInvalidEnum) {
		t.Errorf("marshal invalid ReleaseChannel = %v, want ErrInvalidEnum", err)
	}
}

// TestValidateSHA256BoundaryChars pins the exact hex character boundary: the
// chars immediately outside the accepted ranges ('/' just below '0', ':' just
// above '9', '`' just below 'a', 'g' just above 'f') must all be rejected,
// while the inclusive endpoints '0','9','a','f' are accepted. Off-by-one in the
// range comparisons would fail here.
func TestValidateSHA256BoundaryChars(t *testing.T) {
	for _, c := range []byte{'0', '9', 'a', 'f'} {
		s := strings.Repeat(string(c), 64)
		if err := ValidateSHA256(s); err != nil {
			t.Errorf("ValidateSHA256(all %q) rejected, want accepted: %v", string(c), err)
		}
	}
	for _, c := range []byte{'/', ':', '`', 'g', 'A', 'F'} {
		s := strings.Repeat(string(c), 64)
		if err := ValidateSHA256(s); !errors.Is(err, ErrInvalidSHA256) {
			t.Errorf("ValidateSHA256(all %q) = %v, want ErrInvalidSHA256", string(c), err)
		}
	}
}

// TestHasUpdateAndNoUpdateInvariants pins the UpdateCheckResult constructors'
// contract: NoUpdate yields a nil Update with the false flag (the explicit 204
// semantics), HasUpdate yields a non-nil Update copy carrying the offer with the
// true flag, and HasUpdate stores a copy (mutating the source afterward does not
// alter the stored offer).
func TestHasUpdateAndNoUpdateInvariants(t *testing.T) {
	if nu := NoUpdate(); nu.UpdateAvailable || nu.Update != nil {
		t.Fatalf("NoUpdate() = %+v, want {false, nil}", nu)
	}

	src := UpdateAvailable{ReleaseID: "r1", Version: "1.1.0", URL: "https://x", SHA256: validSHA, Signature: "sig"}
	res := HasUpdate(src)
	if !res.UpdateAvailable || res.Update == nil {
		t.Fatalf("HasUpdate() = %+v, want {true, non-nil}", res)
	}
	if res.Update.ReleaseID != "r1" || res.Update.Version != "1.1.0" {
		t.Fatalf("HasUpdate stored %+v, want fields from src", res.Update)
	}
	// Mutate the source after the call; the stored offer must be unaffected.
	src.Version = "9.9.9"
	if res.Update.Version != "1.1.0" {
		t.Fatal("HasUpdate did not store an independent copy; source mutation leaked in")
	}
}

// TestDeltaUpdateRoundTrip proves the optional Delta offer survives JSON
// round-trip when present and is omitted when nil, so a delta payload offer is
// transmitted faithfully and a legacy (no-delta) offer stays byte-clean.
func TestDeltaUpdateRoundTrip(t *testing.T) {
	withDelta := UpdateAvailable{
		ReleaseID: "r1", Version: "2.0.0", URL: "u", SHA256: validSHA, Signature: "s",
		PayloadProperties: map[string]string{},
		Delta:             &DeltaUpdate{BaseVersion: "1.9.0", URL: "du", Size: 4096, SHA256: validSHA},
	}
	b, err := json.Marshal(withDelta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"delta"`) {
		t.Fatalf("present delta not serialized: %s", b)
	}
	var got UpdateAvailable
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Delta == nil || got.Delta.BaseVersion != "1.9.0" || got.Delta.Size != 4096 {
		t.Fatalf("delta round-trip mismatch: %+v", got.Delta)
	}

	noDelta := UpdateAvailable{ReleaseID: "r1", Version: "2.0.0", URL: "u", SHA256: validSHA, Signature: "s", PayloadProperties: map[string]string{}}
	nb, err := json.Marshal(noDelta)
	if err != nil {
		t.Fatalf("marshal no-delta: %v", err)
	}
	if strings.Contains(string(nb), `"delta"`) {
		t.Fatalf("nil delta must be omitted (omitempty), got: %s", nb)
	}
}

// TestParseHeaderIntRejectsOverflow proves parseHeaderInt rejects a value that
// overflows int64 (strconv.ParseInt fails) rather than silently truncating —
// an overflowing size header is malformed and must surface ErrInvalidValue.
func TestParseHeaderIntRejectsOverflow(t *testing.T) {
	// One past int64 max.
	overflow := "9223372036854775808"
	if _, perr := strconv.ParseInt(overflow, 10, 64); perr == nil {
		t.Fatalf("test premise wrong: %q parsed without overflow", overflow)
	}
	h := map[string]string{
		HeaderFileHash:     "fh",
		HeaderMetadataHash: "mh",
		HeaderFileSize:     overflow,
		HeaderMetadataSize: "1",
	}
	_, err := ParsePayloadProperties(h)
	if !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("overflow FILE_SIZE = %v, want ErrInvalidValue", err)
	}
}

// TestValidateArtifactMetaRejectsWhitespaceOnlyFields proves Board, Version,
// and Signature are rejected when they hold ONLY whitespace (spaces/tabs) —
// a non-empty-but-degenerate value the bare `== ""` completeness check
// cannot catch. The package already recognizes "blank" as a distinct invalid
// class via the unexported isBlank helper (used today only by
// ValidateUpdateCheckRequest.CurrentVersion); this pins the same invariant
// onto ArtifactMeta's three required free-text fields so a board id, version
// string, or signature made of pure whitespace can never silently validate.
// Before the fix, ValidateArtifactMeta(m) with e.g. Board="   " returned nil
// (accepted) because "   " != "" — a real completeness gap.
func TestValidateArtifactMetaRejectsWhitespaceOnlyFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ArtifactMeta)
	}{
		{"blank board spaces", func(m *ArtifactMeta) { m.Board = "   " }},
		{"blank board tab", func(m *ArtifactMeta) { m.Board = "\t" }},
		{"blank board mixed ws", func(m *ArtifactMeta) { m.Board = " \t \t" }},
		{"blank version spaces", func(m *ArtifactMeta) { m.Version = "   " }},
		{"blank version tab", func(m *ArtifactMeta) { m.Version = "\t" }},
		{"blank signature spaces", func(m *ArtifactMeta) { m.Signature = "   " }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := baseArtifactMeta()
			tt.mutate(&m)
			err := ValidateArtifactMeta(m)
			if !errors.Is(err, ErrInvalidValue) {
				t.Fatalf("ValidateArtifactMeta with whitespace-only field = %v, want ErrInvalidValue", err)
			}
		})
	}
	// Sanity: a real non-blank value with incidental internal/edge spacing
	// (e.g. "Orange Pi 5") is NOT blank and must still pass — the fix must
	// reject ONLY all-whitespace strings, not strings that merely contain a
	// space character.
	m := baseArtifactMeta()
	m.Board = "Orange Pi 5"
	if err := ValidateArtifactMeta(m); err != nil {
		t.Fatalf("board containing a real space wrongly rejected: %v", err)
	}
}

// TestIsBlankRecognizesFullASCIIWhitespaceSet proves isBlank treats a string
// as blank when it consists ONLY of standard ASCII whitespace — space, tab,
// newline, carriage return, vertical tab, or form feed (the classic C/POSIX
// isspace() set) — not merely space and tab. Before the fix, isBlank(s)
// returned false for "\n", "\r", "\r\n", "\v", and "\f", so a value that was
// visually/semantically blank (e.g. a lone newline) was treated as
// legitimate non-blank content and passed every isBlank-gated check
// (ValidateUpdateCheckRequest.CurrentVersion, and — after the whitespace-only
// fix above — ArtifactMeta.Board/Version/Signature and
// TelemetryReport.DeviceID/DeploymentID).
func TestIsBlankRecognizesFullASCIIWhitespaceSet(t *testing.T) {
	// isBlank("") is vacuously true (an empty string has no non-whitespace
	// byte) — every call site already gates on non-empty before calling
	// isBlank, so this is intentional, not a defect; included here to pin the
	// vacuous-truth semantics rather than leave it unspecified.
	blank := []string{"", " ", "\t", "\n", "\r", "\r\n", "\v", "\f", "  \t\n\r\v\f  "}
	for _, s := range blank {
		if !isBlank(s) {
			t.Errorf("isBlank(%q) = false, want true (pure ASCII whitespace)", s)
		}
	}
	notBlank := []string{"a", " a ", "1.2.3", "\x00", "x\n"}
	for _, s := range notBlank {
		if isBlank(s) {
			t.Errorf("isBlank(%q) = true, want false (contains non-whitespace)", s)
		}
	}
}

// TestValidateUpdateCheckRequestRejectsNewlineOnlyVersion pins the concrete
// end-to-end consequence of the isBlank gap at the exported-function level: a
// current_version consisting solely of a newline (or CR, or CRLF) MUST be
// rejected exactly like the already-tested all-spaces case, not silently
// accepted as a present, valid version string.
func TestValidateUpdateCheckRequestRejectsNewlineOnlyVersion(t *testing.T) {
	for _, v := range []string{"\n", "\r", "\r\n", "\v", "\f"} {
		err := ValidateUpdateCheckRequest(UpdateCheckRequest{CurrentVersion: v})
		if !errors.Is(err, ErrInvalidValue) {
			t.Errorf("ValidateUpdateCheckRequest(CurrentVersion=%q) = %v, want ErrInvalidValue", v, err)
		}
	}
}

// TestValidateTelemetryReportRejectsWhitespaceOnlyIDs mirrors the above for
// TelemetryReport.DeviceID / DeploymentID: a device_id or deployment_id made
// of pure whitespace is a degenerate value that MUST be rejected, matching
// the completeness bar already applied to ValidateUpdateCheckRequest.
func TestValidateTelemetryReportRejectsWhitespaceOnlyIDs(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*TelemetryReport)
	}{
		{"blank device_id", func(r *TelemetryReport) { r.DeviceID = "   " }},
		{"blank device_id tab", func(r *TelemetryReport) { r.DeviceID = "\t" }},
		{"blank deployment_id", func(r *TelemetryReport) { r.DeploymentID = "   " }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := baseTelemetryReport()
			tt.mutate(&r)
			err := ValidateTelemetryReport(r)
			if !errors.Is(err, ErrInvalidValue) {
				t.Fatalf("ValidateTelemetryReport with whitespace-only id = %v, want ErrInvalidValue", err)
			}
		})
	}
}
