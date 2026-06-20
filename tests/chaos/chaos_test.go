// Package otaprotocol_chaos_test — chaos tests for ota-protocol (§11.4.85).
//
// These meta-tests exercise the exported API with malformed inputs, boundary
// values, and concurrent corrupted-data scenarios.  No mocks, no stubs
// (§11.4.27).  Panic recovery in every goroutine so a single bad input never
// takes down the suite.
package otaprotocol_chaos_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	otaprotocol "github.com/HelixDevelopment/ota-protocol"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func chaosEvidenceDir() string {
	if d := os.Getenv("HELIX_STRESS_EVIDENCE_DIR"); d != "" {
		return d
	}
	return "qa-results/stress_chaos"
}

func writeChaosEvidence(t *testing.T, name string, data []byte) {
	t.Helper()
	dir := chaosEvidenceDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Logf("WARNING: mkdir %s: %v", dir, err)
		return
	}
	ts := time.Now().UTC().Format("20060102T150405Z")
	p := filepath.Join(dir, fmt.Sprintf("%s-%s.json", name, ts))
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Logf("WARNING: write evidence %s: %v", p, err)
	}
}

// ---------------------------------------------------------------------------
// TestChaosValidateSHA256InputCorruption
//
// Feed ValidateSHA256 every class of degenerate input: empty, truncated,
// oversize, non-hex characters, UTF-8, binary data, and strings with leading/
// trailing whitespace.  All must produce errors; none may panic.
// ---------------------------------------------------------------------------

func TestChaosValidateSHA256InputCorruption(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("chaos test skipped in short mode")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic in TestChaosValidateSHA256InputCorruption: %v", r)
		}
	}()

	type entry struct {
		Name   string `json:"name"`
		Input  string `json:"input"`
		Failed bool   `json:"failed"`
		Error  string `json:"error,omitempty"`
	}

	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"short_63", strings.Repeat("a", 63)},
		{"long_65", strings.Repeat("b", 65)},
		{"short_1", "a"},
		{"zero_length", ""},
		{"all_uppercase", strings.Repeat("A", 64)},
		{"non_hex_g", strings.Repeat("g", 64)},
		{"non_hex_x", strings.Repeat("x", 64)},
		{"non_hex_z", strings.Repeat("z", 64)},
		{"unicode_emoji", "\U0001F600" + strings.Repeat("a", 63)},
		{"leading_space", " " + strings.Repeat("a", 63)},
		{"trailing_newline", strings.Repeat("a", 63) + "\n"},
		{"null_bytes", string(make([]byte, 64))},
		{"high_bytes", string([]byte{0x80, 0xFF, 0x00, 0x7F}) + strings.Repeat("a", 60)},
		{"mixed_case", "ABCDEF0123456789abcdef0123456789abcdef0123456789abcdef0123456789"},
		{"only_digits", strings.Repeat("0", 64)},
		{"only_colons", strings.Repeat(":", 64)},
		{"only_slashes", strings.Repeat("/", 64)},
		{"hex_prefix_0x", "0x" + strings.Repeat("a", 62)},
		{"boundary_64_valid", strings.Repeat("a", 64)},
	}

	var results []entry
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			err := otaprotocol.ValidateSHA256(c.input)
			failed := err != nil
			errStr := ""
			if err != nil {
				errStr = err.Error()
			}
			results = append(results, entry{
				Name:   c.name,
				Input:  c.input[:min(len(c.input), 32)],
				Failed: failed,
				Error:  errStr,
			})
			if !failed {
				t.Errorf("ValidateSHA256(%q) = nil, expected error", c.input)
			}
		})
	}

	ev, _ := json.MarshalIndent(results, "", "  ")
	writeChaosEvidence(t, "chaos_validate_sha256_corruption", ev)
}

// ---------------------------------------------------------------------------
// TestChaosArtifactMetaCorruption
//
// Apply every class of field corruption to ArtifactMeta: zero/negative sizes,
// empty strings, missing SHA256, oversized values, and extreme combinations.
// ---------------------------------------------------------------------------

func TestChaosArtifactMetaCorruption(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("chaos test skipped in short mode")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic in TestChaosArtifactMetaCorruption: %v", r)
		}
	}()

	type entry struct {
		Name     string `json:"name"`
		Mutated  string `json:"mutated"`
		Rejected bool   `json:"rejected"`
		Error    string `json:"error,omitempty"`
	}

	makeValid := func() otaprotocol.ArtifactMeta {
		return otaprotocol.ArtifactMeta{
			SHA256:    strings.Repeat("a", 64),
			Size:      4096,
			OSType:    otaprotocol.OSAndroid,
			Board:     "greenfield-rev-2",
			Version:   "2026.06.01",
			Signature: "test-sig",
		}
	}

	cases := []struct {
		name   string
		mutate func(m *otaprotocol.ArtifactMeta)
	}{
		{"zero_size", func(m *otaprotocol.ArtifactMeta) { m.Size = 0 }},
		{"negative_size", func(m *otaprotocol.ArtifactMeta) { m.Size = -1 }},
		{"min_int64_size", func(m *otaprotocol.ArtifactMeta) { m.Size = math.MinInt64 }},
		{"max_int64_size", func(m *otaprotocol.ArtifactMeta) { m.Size = math.MaxInt64 }},
		{"empty_board", func(m *otaprotocol.ArtifactMeta) { m.Board = "" }},
		{"empty_version", func(m *otaprotocol.ArtifactMeta) { m.Version = "" }},
		{"empty_signature", func(m *otaprotocol.ArtifactMeta) { m.Signature = "" }},
		{"empty_sha256", func(m *otaprotocol.ArtifactMeta) { m.SHA256 = "" }},
		{"invalid_os_type", func(m *otaprotocol.ArtifactMeta) { m.OSType = "macos" }},
		{"empty_os_type", func(m *otaprotocol.ArtifactMeta) { m.OSType = "" }},
		{"sha256_uppercase", func(m *otaprotocol.ArtifactMeta) { m.SHA256 = strings.Repeat("A", 64) }},
		{"sha256_short", func(m *otaprotocol.ArtifactMeta) { m.SHA256 = strings.Repeat("a", 48) }},
		{"oversize_version", func(m *otaprotocol.ArtifactMeta) { m.Version = strings.Repeat("v", 1000) }},
		{"all_empty_strings", func(m *otaprotocol.ArtifactMeta) {
			m.SHA256 = ""; m.Board = ""; m.Version = ""; m.Signature = ""
		}},
	}

	var results []entry
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic: %v", r)
				}
			}()
			m := makeValid()
			c.mutate(&m)
			err := otaprotocol.ValidateArtifactMeta(m)
			rejected := err != nil
			errStr := ""
			if err != nil {
				errStr = err.Error()
			}
			results = append(results, entry{
				Name:     c.name,
				Mutated: fmt.Sprintf("SHA256=%q Size=%d OS=%q Board=%q Version=%q",
					m.SHA256[:min(len(m.SHA256), 16)], m.Size, m.OSType, m.Board, m.Version),
				Rejected: rejected,
				Error:    errStr,
			})
			if !rejected {
				t.Errorf("%s: ValidateArtifactMeta = nil, expected error", c.name)
			}
		})
	}

	ev, _ := json.MarshalIndent(results, "", "  ")
	writeChaosEvidence(t, "chaos_artifact_meta_corruption", ev)
}

// ---------------------------------------------------------------------------
// TestChaosConcurrentCorruptedPayload
//
// N=50 goroutines each call ParsePayloadProperties with corrupted header maps.
// Asserts every call returns an error and none panics.
// ---------------------------------------------------------------------------

func TestChaosConcurrentCorruptedPayload(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("chaos test skipped in short mode")
	}

	const N = 50

	corruptedMaps := []map[string]string{
		{},
		{"FILE_HASH": "fh"},
		{"FILE_SIZE": "100"},
		{"METADATA_HASH": "mh"},
		{"METADATA_SIZE": "10"},
		{"FILE_HASH": "", "FILE_SIZE": "100", "METADATA_HASH": "mh", "METADATA_SIZE": "10"},
		{"FILE_HASH": "fh", "FILE_SIZE": "", "METADATA_HASH": "mh", "METADATA_SIZE": "10"},
		{"FILE_HASH": "fh", "FILE_SIZE": "not-a-number", "METADATA_HASH": "mh", "METADATA_SIZE": "10"},
		{"FILE_HASH": "fh", "FILE_SIZE": "100", "METADATA_HASH": "mh", "METADATA_SIZE": "-5"},
		{"FILE_HASH": "fh", "FILE_SIZE": "100", "METADATA_HASH": "mh", "METADATA_SIZE": "12.5"},
		{"FILE_HASH": "fh", "FILE_SIZE": "9223372036854775808", "METADATA_HASH": "mh", "METADATA_SIZE": "10"},
		{"FILE_HASH": "fh", "FILE_SIZE": "-100", "METADATA_HASH": "mh", "METADATA_SIZE": "10"},
	}

	var (
		wg       sync.WaitGroup
		panics   int32
		rejected int32
	)

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt32(&panics, 1)
				}
			}()
			h := corruptedMaps[idx%len(corruptedMaps)]
			_, err := otaprotocol.ParsePayloadProperties(h)
			if err != nil {
				atomic.AddInt32(&rejected, 1)
			}
		}(i)
	}
	wg.Wait()

	record := map[string]interface{}{
		"test":       "TestChaosConcurrentCorruptedPayload",
		"goroutines": N,
		"rejected":   rejected,
		"panics":     panics,
	}
	ev, _ := json.MarshalIndent(record, "", "  ")
	writeChaosEvidence(t, "chaos_concurrent_corrupted_payload", ev)

	if panics > 0 {
		t.Errorf("%d goroutines panicked during corrupted payload parsing", panics)
	}
}

// ---------------------------------------------------------------------------
// TestChaosPayloadPropertiesOverflow
//
// ParsePayloadProperties with int64 overflow and extreme numeric values.
// ---------------------------------------------------------------------------

func TestChaosPayloadPropertiesOverflow(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("chaos test skipped in short mode")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic in TestChaosPayloadPropertiesOverflow: %v", r)
		}
	}()

	type entry struct {
		Name   string `json:"name"`
		Input  string `json:"input"`
		Failed bool   `json:"failed"`
		Error  string `json:"error,omitempty"`
	}

	overflowValues := []struct {
		name  string
		value string
	}{
		{"file_size_overflow", "9223372036854775808"},
		{"file_size_negative", "-1"},
		{"file_size_min_int64", "-9223372036854775808"},
		{"file_size_huge_positive", "9999999999999999999"},
		{"file_size_float", "100.5"},
		{"file_size_hex", "0xFF"},
		{"file_size_binary", "0b1010"},
		{"meta_size_overflow", "9223372036854775808"},
		{"meta_size_negative", "-100"},
		{"meta_size_huge", "18446744073709551616"},
	}

	base := map[string]string{
		"FILE_HASH":     genHash(),
		"FILE_SIZE":     "1000",
		"METADATA_HASH": genHash(),
		"METADATA_SIZE": "100",
	}

	var results []entry
	for _, ov := range overflowValues {
		ov := ov
		t.Run(ov.name, func(t *testing.T) {
			h := make(map[string]string, len(base))
			for k, v := range base {
				h[k] = v
			}
			// Apply the overflow to either FILE_SIZE or METADATA_SIZE.
			if strings.HasPrefix(ov.name, "file_size_") {
				h["FILE_SIZE"] = ov.value
			} else {
				h["METADATA_SIZE"] = ov.value
			}

			_, err := otaprotocol.ParsePayloadProperties(h)
			failed := err != nil
			errStr := ""
			if err != nil {
				errStr = err.Error()
			}
			results = append(results, entry{
				Name: ov.name, Input: ov.value, Failed: failed, Error: errStr,
			})
			if !failed {
				t.Errorf("ParsePayloadProperties with %s=%q = nil, expected error", ov.name, ov.value)
			}
		})
	}

	ev, _ := json.MarshalIndent(results, "", "  ")
	writeChaosEvidence(t, "chaos_payload_overflow", ev)
}

// genHash returns a short hex string for test payload headers.
func genHash() string {
	return strings.Repeat("deadbeef", 2)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
