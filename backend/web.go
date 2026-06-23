package main

import (
	"embed"
	"io/fs"
)

// web/dist holds the built Vue frontend (frontend/'s vite.config.js
// writes its build output directly here via build.outDir, so a plain
// `npm run build` in frontend/ is all it takes to refresh this). A
// placeholder index.html is committed so the embed directive - and
// therefore `go build`/`go test` - never fails even before the real
// frontend has been built once.
//
//go:embed web/dist
var embeddedFrontend embed.FS

// frontendFS strips the "web/dist" prefix so the embedded files are
// served at "/", not "/web/dist/".
func frontendFS() fs.FS {
	sub, err := fs.Sub(embeddedFrontend, "web/dist")
	if err != nil {
		// Can't happen: web/dist is a literal path embedded above.
		panic(err)
	}
	return sub
}
