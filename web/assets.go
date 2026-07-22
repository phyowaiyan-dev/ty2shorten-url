package web

import "embed"

// FS contains templates and static files embedded into the application binary.
//
//go:embed templates/**/*.html static/**/* content/*.md
var FS embed.FS
