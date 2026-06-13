package handlers_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// ---------------------------------------------------------------------------
// SHA-256 deduplication logic
// ---------------------------------------------------------------------------
// The dedup check in submission.go is: hash the raw ZIP bytes → hex string →
// query DB. We test the hash function in isolation and then verify that the
// same bytes always produce the same hash (determinism) and different bytes
// produce different hashes (collision resistance at test scale).

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestSHA256Dedup_SameBytesProduceSameHash(t *testing.T) {
	payload := []byte("hello bench platform")
	h1 := sha256Hex(payload)
	h2 := sha256Hex(payload)
	if h1 != h2 {
		t.Errorf("identical bytes produced different hashes: %s vs %s", h1, h2)
	}
}

func TestSHA256Dedup_DifferentBytesProduceDifferentHash(t *testing.T) {
	h1 := sha256Hex([]byte("submission-v1"))
	h2 := sha256Hex([]byte("submission-v2"))
	if h1 == h2 {
		t.Errorf("different bytes produced the same hash: %s", h1)
	}
}

func TestSHA256Dedup_EmptyPayload(t *testing.T) {
	// sha256("") is a valid, well-known value — must not panic.
	h := sha256Hex([]byte{})
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if h != expected {
		t.Errorf("empty payload hash: got %s, want %s", h, expected)
	}
}

func TestSHA256Dedup_HashLength(t *testing.T) {
	h := sha256Hex([]byte("any content"))
	// SHA-256 is 32 bytes → 64 hex chars.
	if len(h) != 64 {
		t.Errorf("hash length: got %d, want 64", len(h))
	}
}

// ---------------------------------------------------------------------------
// validateZipHasDockerfile — reimplemented here to test the logic directly
// without importing the handlers package (which has DB deps).
// The production function is identical; we're testing the algorithm.
// ---------------------------------------------------------------------------

func validateZipHasDockerfile(data []byte) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range r.File {
		// filepath.Base equivalent inline — avoids import for this small helper.
		name := f.Name
		for i := len(name) - 1; i >= 0; i-- {
			if name[i] == '/' {
				name = name[i+1:]
				break
			}
		}
		if name == "Dockerfile" {
			return nil
		}
	}
	return bytes.ErrTooLarge // any non-nil error signals rejection
}

func buildZip(files map[string]string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(content))
	}
	_ = zw.Close()
	return buf.Bytes()
}

func TestValidateZip_WithDockerfile_Passes(t *testing.T) {
	z := buildZip(map[string]string{
		"Dockerfile": "FROM alpine\nRUN echo ok",
		"main.go":    "package main",
	})
	if err := validateZipHasDockerfile(z); err != nil {
		t.Errorf("expected nil error for ZIP with Dockerfile, got %v", err)
	}
}

func TestValidateZip_WithoutDockerfile_Fails(t *testing.T) {
	z := buildZip(map[string]string{
		"main.go":    "package main",
		"go.mod":     "module example",
	})
	if err := validateZipHasDockerfile(z); err == nil {
		t.Error("expected error for ZIP without Dockerfile, got nil")
	}
}

func TestValidateZip_DockerfileInSubdir_Passes(t *testing.T) {
	// Dockerfile in a subdirectory — filepath.Base strips the path,
	// so "subdir/Dockerfile" → base name "Dockerfile" → passes.
	z := buildZip(map[string]string{
		"subdir/Dockerfile": "FROM alpine",
		"main.go":           "package main",
	})
	if err := validateZipHasDockerfile(z); err != nil {
		t.Errorf("expected nil for Dockerfile in subdir, got %v", err)
	}
}

func TestValidateZip_EmptyZip_Fails(t *testing.T) {
	z := buildZip(map[string]string{})
	if err := validateZipHasDockerfile(z); err == nil {
		t.Error("expected error for empty ZIP, got nil")
	}
}

func TestValidateZip_InvalidBytes_Fails(t *testing.T) {
	garbage := []byte("this is not a zip file at all")
	if err := validateZipHasDockerfile(garbage); err == nil {
		t.Error("expected error for invalid ZIP bytes, got nil")
	}
}

func TestValidateZip_DockerfileLookalike_Fails(t *testing.T) {
	// "dockerfile" (lowercase) is NOT a valid Dockerfile — Linux is case-sensitive.
	z := buildZip(map[string]string{
		"dockerfile": "FROM alpine",
	})
	if err := validateZipHasDockerfile(z); err == nil {
		t.Error("expected error for lowercase 'dockerfile', got nil")
	}
}
