package assets

import "embed"

// FS contains built-in project icons and web app metadata.
//
//go:embed *.ico *.png *.svg *.webmanifest
var FS embed.FS
