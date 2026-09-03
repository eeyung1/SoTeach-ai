// Package web embeds the static Stage 2 web client so the Go server can serve
// it from a single self-contained binary, independent of the working
// directory (workingReadme §3.5: a lightweight, text-first browser client
// over the Stage 1 API; §8 Stage 2).
package web

import "embed"

// Files holds the web client's static assets.
//
//go:embed index.html app.js style.css
var Files embed.FS
