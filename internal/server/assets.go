package server

import (
	"embed"
)

// StaticAssets embeds all static frontend files (HTML, CSS, JS) into the compiled Go binary.
//
//go:embed static/*
var StaticAssets embed.FS
