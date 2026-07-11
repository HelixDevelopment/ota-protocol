package otaprotocol

import (
	"errors"
	"testing"
)

// TestParsePayloadPropertiesRejectsNegativeSizes is the permanent regression
// guard for HC-3: ParsePayloadProperties previously parsed FILE_SIZE /
// METADATA_SIZE via strconv.ParseInt without checking sign, so a header map
// carrying a syntactically valid but negative size string (e.g. "-999")
// parsed to a PayloadProperties{FileSize: -999, ...} with a nil error --
// even though the very same file's PayloadProperties.Validate() explicitly
// rejects FileSize < 0 / MetadataSize < 0 with ErrInvalidValue. Parse was
// permissive where Validate was strict (a §11.4.108 SOURCE-layer divergence
// between the parse path and the validate path for the identical field).
//
// This test asserts ParsePayloadProperties itself now rejects a negative
// size at parse time, matching Validate's invariant, plus a positive
// control proving a legitimate non-negative header set still parses
// cleanly (so the fix does not over-reject valid input).
func TestParsePayloadPropertiesRejectsNegativeSizes(t *testing.T) {
	good := samplePayloadProperties().Headers()

	tests := []struct {
		name    string
		headers map[string]string
		wantErr error
	}{
		{"negative file size", with(good, HeaderFileSize, "-999"), ErrInvalidValue},
		{"negative metadata size", with(good, HeaderMetadataSize, "-1"), ErrInvalidValue},
		{"negative both sizes", with(with(good, HeaderFileSize, "-42"), HeaderMetadataSize, "-7"), ErrInvalidValue},
		// Positive control: a valid non-negative header set MUST still parse
		// with no error -- the fix must not over-reject legitimate input.
		{"non-negative sizes still parse OK", good, nil},
		// Boundary: zero is a valid non-negative size, MUST NOT be rejected.
		{"zero file size is not negative", with(good, HeaderFileSize, "0"), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ParsePayloadProperties(tt.headers)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("got error %v, want nil", err)
				}
				if p.FileSize < 0 || p.MetadataSize < 0 {
					t.Fatalf("parsed negative size on the no-error path: FileSize=%d MetadataSize=%d", p.FileSize, p.MetadataSize)
				}
				return
			}
			if err == nil {
				t.Fatalf("got nil error, want error wrapping %v (parsed %+v)", tt.wantErr, p)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want error wrapping %v", err, tt.wantErr)
			}
		})
	}
}

// TestParsePayloadPropertiesNegativeSizeMatchesValidate proves
// ParsePayloadProperties and PayloadProperties.Validate now agree on the
// non-negative-size invariant for the same input: constructing a
// PayloadProperties struct directly with a negative FileSize and running it
// through Validate() must fail with ErrInvalidValue, exactly as
// ParsePayloadProperties now fails at parse time for the equivalent header.
// This is the parse-vs-validate divergence closure the HC-3 fix targets.
func TestParsePayloadPropertiesNegativeSizeMatchesValidate(t *testing.T) {
	good := samplePayloadProperties().Headers()

	_, parseErr := ParsePayloadProperties(with(good, HeaderFileSize, "-1"))
	if !errors.Is(parseErr, ErrInvalidValue) {
		t.Fatalf("ParsePayloadProperties: got %v, want error wrapping ErrInvalidValue", parseErr)
	}

	direct := samplePayloadProperties()
	direct.FileSize = -1
	validateErr := direct.Validate()
	if !errors.Is(validateErr, ErrInvalidValue) {
		t.Fatalf("Validate: got %v, want error wrapping ErrInvalidValue", validateErr)
	}
}
