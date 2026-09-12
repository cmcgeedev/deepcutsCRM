// Package storage saves proof images under a local directory. S3 can replace it later.
package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Local struct{ Dir string }

func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Local{Dir: dir}, nil
}

func sniff(data []byte) (ext, contentType string, ok bool) {
	switch {
	case bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}):
		return "png", "image/png", true
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return "jpg", "image/jpeg", true
	}
	return "", "", false
}

func (l *Local) Save(kind string, data []byte) (string, error) {
	ext, _, ok := sniff(data)
	if !ok {
		return "", errors.New("proof must be a PNG or JPEG image")
	}
	if strings.ContainsAny(kind, "/\\.") {
		return "", errors.New("invalid kind")
	}
	if err := os.MkdirAll(filepath.Join(l.Dir, kind), 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s.%s", uuid.NewString(), ext)
	if err := os.WriteFile(filepath.Join(l.Dir, kind, name), data, 0o644); err != nil {
		return "", err
	}
	return kind + "/" + name, nil
}

func (l *Local) Open(ref string) (io.ReadCloser, string, error) {
	if filepath.IsAbs(ref) || strings.Contains(ref, "..") {
		return nil, "", errors.New("invalid ref")
	}
	ct := "application/octet-stream"
	switch filepath.Ext(ref) {
	case ".png":
		ct = "image/png"
	case ".jpg":
		ct = "image/jpeg"
	}
	f, err := os.Open(filepath.Join(l.Dir, filepath.FromSlash(ref)))
	if err != nil {
		return nil, "", err
	}
	return f, ct, nil
}
