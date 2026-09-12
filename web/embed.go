// Package web embeds the built React app. Run `make build-web` before building the binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var Dist embed.FS

func FS() (fs.FS, error) { return fs.Sub(Dist, "dist") }
