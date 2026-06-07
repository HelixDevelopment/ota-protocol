package otaprotocol

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

// zeroTime returns the zero time.Time for use in table tests.
func zeroTime() time.Time { return time.Time{} }

func samplePayloadProperties() PayloadProperties {
	return PayloadProperties{
		FileHash:     "ZmlsZWhhc2g=",
		FileSize:     379074366,
		MetadataHash: "bWV0YWhhc2g=",
		MetadataSize: 46866,
	}
}

func TestPayloadPropertiesHeaders(t *testing.T) {
	p := samplePayloadProperties()
	h := p.Headers()
	want := map[string]string{
		HeaderFileHash:     "ZmlsZWhhc2g=",
		HeaderFileSize:     "379074366",
		HeaderMetadataHash: "bWV0YWhhc2g=",
		HeaderMetadataSize: "46866",
	}
	if !reflect.DeepEqual(h, want) {
		t.Errorf("Headers() = %v, want %v", h, want)
	}
}

func TestPayloadPropertiesHeadersRoundTrip(t *testing.T) {
	in := samplePayloadProperties()
	got, err := ParsePayloadProperties(in.Headers())
	if err != nil {
		t.Fatalf("ParsePayloadProperties: %v", err)
	}
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
}

func TestParsePayloadProperties(t *testing.T) {
	good := samplePayloadProperties().Headers()

	tests := []struct {
		name    string
		headers map[string]string
		wantErr error
	}{
		{"valid", good, nil},
		{"missing file hash", without(good, HeaderFileHash), ErrMissingPayloadHeader},
		{"missing metadata hash", without(good, HeaderMetadataHash), ErrMissingPayloadHeader},
		{"missing file size", without(good, HeaderFileSize), ErrMissingPayloadHeader},
		{"missing metadata size", without(good, HeaderMetadataSize), ErrMissingPayloadHeader},
		{"non-int file size", with(good, HeaderFileSize, "abc"), ErrInvalidValue},
		{"non-int metadata size", with(good, HeaderMetadataSize, "12.5"), ErrInvalidValue},
		{"empty map", map[string]string{}, ErrMissingPayloadHeader},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePayloadProperties(tt.headers)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("got %v, want nil", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestPayloadPropertiesValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*PayloadProperties)
		wantErr error
	}{
		{"valid", func(p *PayloadProperties) {}, nil},
		{"zero sizes ok", func(p *PayloadProperties) { p.FileSize = 0; p.MetadataSize = 0 }, nil},
		{"empty file hash", func(p *PayloadProperties) { p.FileHash = "" }, ErrMissingField},
		{"empty metadata hash", func(p *PayloadProperties) { p.MetadataHash = "" }, ErrMissingField},
		{"negative file size", func(p *PayloadProperties) { p.FileSize = -1 }, ErrInvalidValue},
		{"negative metadata size", func(p *PayloadProperties) { p.MetadataSize = -1 }, ErrInvalidValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := samplePayloadProperties()
			tt.mutate(&p)
			err := p.Validate()
			if tt.wantErr == nil && err != nil {
				t.Fatalf("got %v, want nil", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestPayloadPropertiesJSONRoundTrip confirms the struct serializes with the
// SCREAMING_SNAKE_CASE JSON keys from the OpenAPI schema.
func TestPayloadPropertiesJSONRoundTrip(t *testing.T) {
	in := samplePayloadProperties()
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got PayloadProperties
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal %s: %v", b, err)
	}
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	for _, k := range []string{HeaderFileHash, HeaderFileSize, HeaderMetadataHash, HeaderMetadataSize} {
		if _, ok := raw[k]; !ok {
			t.Errorf("JSON missing key %q in %s", k, b)
		}
	}
}

// without returns a copy of m without key k.
func without(m map[string]string, k string) map[string]string {
	out := make(map[string]string, len(m))
	for kk, vv := range m {
		if kk != k {
			out[kk] = vv
		}
	}
	return out
}

// with returns a copy of m with key k set to v.
func with(m map[string]string, k, v string) map[string]string {
	out := make(map[string]string, len(m))
	for kk, vv := range m {
		out[kk] = vv
	}
	out[k] = v
	return out
}
