package otaprotocol

import (
	"errors"
	"testing"
)

// FuzzParsePayloadProperties fuzzes the update_engine payload_properties header
// parse path. The four mandatory headers (FILE_HASH/FILE_SIZE/METADATA_HASH/
// METADATA_SIZE) are attacker-influenceable when a payload-properties blob is
// received and decoded into a string map. ParsePayloadProperties runs
// strconv.ParseInt on the size values. Property: any header map must yield a
// PayloadProperties + nil error OR an error — never a panic — and a successful
// parse must round-trip its size strings back to the same integers.
func FuzzParsePayloadProperties(f *testing.F) {
	// Seed with combinations of the four keys carrying valid + edge values.
	type kv struct{ fh, fs, mh, ms string }
	seeds := []kv{
		{"abc", "1024", "def", "256"},                            // valid
		{"", "0", "", "0"},                                       // empty hashes
		{"h", "-1", "h", "-1"},                                   // negative sizes
		{"h", "9223372036854775807", "h", "9223372036854775807"}, // int64 max
		{"h", "9223372036854775808", "h", "0"},                   // overflow
		{"h", "0x10", "h", "0"},                                  // hex-looking
		{"h", "   12  ", "h", "0"},                               // padded
		{"h", "", "h", "0"},                                      // empty size
		{"h", "+5", "h", "-0"},                                   // signs
		{"h", "１２３", "h", "0"},                                   // fullwidth digits
	}
	for _, s := range seeds {
		f.Add(s.fh, s.fs, s.mh, s.ms)
	}

	f.Fuzz(func(t *testing.T, fileHash, fileSize, metaHash, metaSize string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ParsePayloadProperties panicked on (%q,%q,%q,%q): %v",
					fileHash, fileSize, metaHash, metaSize, r)
			}
		}()

		h := map[string]string{
			HeaderFileHash:     fileHash,
			HeaderFileSize:     fileSize,
			HeaderMetadataHash: metaHash,
			HeaderMetadataSize: metaSize,
		}
		p, err := ParsePayloadProperties(h)
		if err != nil {
			_ = errors.Is(err, ErrInvalidValue)
			_ = errors.Is(err, ErrMissingPayloadHeader)
			return
		}
		// Round-trip: a parsed value rendered back to headers must re-parse to
		// the same sizes (no lossy parse). Headers()/Validate() must not panic.
		_ = p.Validate()
		rt := p.Headers()
		p2, err2 := ParsePayloadProperties(rt)
		if err2 != nil {
			t.Fatalf("re-parse of rendered headers failed: %v (orig %q/%q)", err2, fileSize, metaSize)
		}
		if p2.FileSize != p.FileSize || p2.MetadataSize != p.MetadataSize {
			t.Fatalf("round-trip size mismatch: %+v vs %+v", p, p2)
		}
	})
}

// FuzzValidateSHA256 fuzzes the SHA-256 digest validator. Artifact metadata and
// payload hashes arrive as untrusted strings; ValidateSHA256 is the gate. It
// must classify every string as valid-or-error without panicking, and must NOT
// accept any string that is not exactly 64 lowercase-hex chars.
func FuzzValidateSHA256(f *testing.F) {
	f.Add("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855") // valid (empty-string digest)
	f.Add("")
	f.Add("ABCDEF")
	f.Add("zz")
	f.Add("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b85")   // 63
	f.Add("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b8555") // 65
	f.Add("ＡＢＣ")                                                               // fullwidth

	f.Fuzz(func(t *testing.T, s string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ValidateSHA256 panicked on %q: %v", s, r)
			}
		}()
		err := ValidateSHA256(s)
		// Cross-check the validator against an independent definition: valid IFF
		// 64 chars all in [0-9a-f]. A disagreement is a real correctness bug.
		want := len(s) == 64
		if want {
			for i := 0; i < len(s); i++ {
				c := s[i]
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					want = false
					break
				}
			}
		}
		if want && err != nil {
			t.Fatalf("ValidateSHA256(%q) rejected a well-formed digest: %v", s, err)
		}
		if !want && err == nil {
			t.Fatalf("ValidateSHA256(%q) accepted a malformed digest", s)
		}
	})
}
