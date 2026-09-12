package storage

import (
	"io"
	"testing"
)

var png = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0}
var jpg = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0}

func TestSaveAndOpen(t *testing.T) {
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ref, err := l.Save("proofs", png)
	if err != nil {
		t.Fatal(err)
	}
	rc, ct, err := l.Open(ref)
	if err != nil || ct != "image/png" {
		t.Fatalf("open: %v %s", err, ct)
	}
	b, _ := io.ReadAll(rc)
	rc.Close()
	if string(b) != string(png) {
		t.Fatal("content mismatch")
	}
	ref2, _ := l.Save("proofs", jpg)
	if _, ct, _ := l.Open(ref2); ct != "image/jpeg" {
		t.Fatalf("jpeg content type: %s", ct)
	}
	if _, err := l.Save("proofs", []byte("not an image")); err == nil {
		t.Fatal("non-image must be rejected")
	}
	for _, bad := range []string{"../etc/passwd", "/etc/passwd", "proofs/../../x.png"} {
		if _, _, err := l.Open(bad); err == nil {
			t.Fatalf("path %q must be rejected", bad)
		}
	}
}
