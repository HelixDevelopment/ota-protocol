package otaprotocol

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"
)

// evidenceDir returns the directory for captured-evidence files, creating it if
// needed. The caller may override the default location via HELIX_STRESS_EVIDENCE_DIR.
func evidenceDir() string {
	dir := os.Getenv("HELIX_STRESS_EVIDENCE_DIR")
	if dir == "" {
		dir = "qa-results/stress_chaos"
	}
	return dir
}

// writeEvidence encodes a single JSON record as JSONL to a timestamped evidence
// file under evidenceDir()/<test>-<ts>.jsonl. Failures are silently tolerated
// (best-effort — the test itself is the primary signal).
func writeEvidence(testName string, record interface{}) {
	dir := evidenceDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	ts := time.Now().UTC().Format("20060102T150405Z")
	p := filepath.Join(dir, fmt.Sprintf("%s-%s.jsonl", testName, ts))
	f, err := os.Create(p)
	if err != nil {
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	_ = enc.Encode(record)
}

// percentiles computes p50, p95, and p99 from a sorted time.Duration slice.
func percentiles(durations []time.Duration) (p50, p95, p99 time.Duration) {
	n := len(durations)
	if n == 0 {
		return 0, 0, 0
	}
	sorted := make([]time.Duration, n)
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	p50 = sorted[n*50/100]
	p95 = sorted[n*95/100]
	p99 = sorted[n*99/100]
	return
}

// genValidSHA256 generates a random valid lowercase-hex SHA-256 digest.
func genValidSHA256() string {
	const hex = "0123456789abcdef"
	b := make([]byte, 64)
	for i := range b {
		b[i] = hex[rand.Intn(len(hex))]
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// Stress: concurrent ValidateSHA256
// ---------------------------------------------------------------------------

func TestStressConcurrentValidateSHA256(t *testing.T) {
	if testing.Short() {
		t.Skip("stress test skipped in short mode")
	}

	const N = 100
	validHashes := make([]string, N)
	for i := range validHashes {
		validHashes[i] = genValidSHA256()
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make([]time.Duration, N)
	)

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			start := time.Now()
			err := ValidateSHA256(validHashes[idx])
			elapsed := time.Since(start)

			mu.Lock()
			results[idx] = elapsed
			mu.Unlock()

			if err != nil {
				t.Errorf("goroutine %d: ValidateSHA256(%q...) = %v", idx, validHashes[idx][:8], err)
			}
		}(i)
	}
	wg.Wait()

	p50, p95, p99 := percentiles(results)
	record := map[string]interface{}{
		"test":     "TestStressConcurrentValidateSHA256",
		"N":        N,
		"p50_ns":   p50.Nanoseconds(),
		"p95_ns":   p95.Nanoseconds(),
		"p99_ns":   p99.Nanoseconds(),
		"all_pass": !t.Failed(),
	}
	writeEvidence("ValidateSHA256", record)
	t.Logf("ValidateSHA256 concurrent N=%d: p50=%v p95=%v p99=%v", N, p50, p95, p99)
}

// ---------------------------------------------------------------------------
// Stress: concurrent ValidateArtifactMeta with OSType cross-product
// ---------------------------------------------------------------------------

func TestStressConcurrentValidateArtifactMeta(t *testing.T) {
	if testing.Short() {
		t.Skip("stress test skipped in short mode")
	}

	const N = 100
	ostypes := []OSType{OSAndroid, OSLinux, OSWindows, OSOther}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make([]time.Duration, N)
	)

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			m := baseArtifactMeta()
			m.OSType = ostypes[rand.Intn(len(ostypes))]

			start := time.Now()
			err := ValidateArtifactMeta(m)
			elapsed := time.Since(start)

			mu.Lock()
			results[idx] = elapsed
			mu.Unlock()

			if err != nil {
				t.Errorf("goroutine %d: ValidateArtifactMeta = %v", idx, err)
			}
		}(i)
	}
	wg.Wait()

	p50, p95, p99 := percentiles(results)
	record := map[string]interface{}{
		"test":     "TestStressConcurrentValidateArtifactMeta",
		"N":        N,
		"p50_ns":   p50.Nanoseconds(),
		"p95_ns":   p95.Nanoseconds(),
		"p99_ns":   p99.Nanoseconds(),
		"all_pass": !t.Failed(),
	}
	writeEvidence("ValidateArtifactMeta", record)
	t.Logf("ValidateArtifactMeta concurrent N=%d: p50=%v p95=%v p99=%v", N, p50, p95, p99)
}

// ---------------------------------------------------------------------------
// Stress: concurrent ParsePayloadProperties
// ---------------------------------------------------------------------------

func TestStressConcurrentParsePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("stress test skipped in short mode")
	}

	const N = 100

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make([]time.Duration, N)
	)

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			h := map[string]string{
				HeaderFileHash:     genValidSHA256()[:24],
				HeaderFileSize:     fmt.Sprintf("%d", 1000+idx),
				HeaderMetadataHash: genValidSHA256()[:16],
				HeaderMetadataSize: fmt.Sprintf("%d", 100+idx),
			}

			start := time.Now()
			p, err := ParsePayloadProperties(h)
			elapsed := time.Since(start)

			mu.Lock()
			results[idx] = elapsed
			mu.Unlock()

			if err != nil {
				t.Errorf("goroutine %d: ParsePayloadProperties = %v", idx, err)
				return
			}
			if p.FileSize != int64(1000+idx) {
				t.Errorf("goroutine %d: FileSize = %d, want %d", idx, p.FileSize, 1000+idx)
			}
		}(i)
	}
	wg.Wait()

	p50, p95, p99 := percentiles(results)
	record := map[string]interface{}{
		"test":     "TestStressConcurrentParsePayload",
		"N":        N,
		"p50_ns":   p50.Nanoseconds(),
		"p95_ns":   p95.Nanoseconds(),
		"p99_ns":   p99.Nanoseconds(),
		"all_pass": !t.Failed(),
	}
	writeEvidence("ParsePayload", record)
	t.Logf("ParsePayloadProperties concurrent N=%d: p50=%v p95=%v p99=%v", N, p50, p95, p99)
}

// ---------------------------------------------------------------------------
// Stress: concurrent enum Valid() calls with random values
// ---------------------------------------------------------------------------

func TestStressConcurrentEnumValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("stress test skipped in short mode")
	}

	const N = 100

	type enumCall struct {
		name string
		fn   func() bool
	}
	calls := make([]enumCall, 0, 20)

	for _, v := range osTypeValues {
		v := v
		calls = append(calls, enumCall{"OSType.Valid(" + v.String() + ")", v.Valid})
	}
	for _, s := range []string{"macos", "solaris", "ios", "", "androidx"} {
		s := s
		calls = append(calls, enumCall{"OSType.Valid(" + s + ")", OSType(s).Valid})
	}
	for _, v := range artifactTypeValues {
		v := v
		calls = append(calls, enumCall{"ArtifactType.Valid(" + v.String() + ")", v.Valid})
	}
	calls = append(calls, enumCall{"ArtifactType.Valid(patch)", ArtifactType("patch").Valid})
	for _, v := range uploadStatusValues {
		v := v
		calls = append(calls, enumCall{"UploadStatus.Valid(" + v.String() + ")", v.Valid})
	}
	calls = append(calls, enumCall{"UploadStatus.Valid(queued)", UploadStatus("queued").Valid})
	for _, v := range telemetryEventValues {
		v := v
		calls = append(calls, enumCall{"TelemetryEvent.Valid(" + v.String() + ")", v.Valid})
	}
	calls = append(calls, enumCall{"TelemetryEvent.Valid(idle)", TelemetryEvent("idle").Valid})

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make([]time.Duration, N)
	)

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			call := calls[rand.Intn(len(calls))]
			start := time.Now()
			_ = call.fn()
			elapsed := time.Since(start)

			mu.Lock()
			results[idx] = elapsed
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	p50, p95, p99 := percentiles(results)
	record := map[string]interface{}{
		"test":   "TestStressConcurrentEnumValidation",
		"N":      N,
		"p50_ns": p50.Nanoseconds(),
		"p95_ns": p95.Nanoseconds(),
		"p99_ns": p99.Nanoseconds(),
	}
	writeEvidence("EnumValidation", record)
	t.Logf("Enum Valid() concurrent N=%d: p50=%v p95=%v p99=%v", N, p50, p95, p99)
}
