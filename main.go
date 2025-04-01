//go:build !next

package main

import (
	"embed"
	// Embed tzdata in binary.
	//
	// See https://github.com/AdGuardPrivate/AdGuardPrivate/issues/6758
	_ "time/tzdata"

	"github.com/AdGuardPrivate/AdGuardPrivate/internal/home"
)

// Embed the prebuilt client here since we strive to keep .go files inside the
// internal directory and the embed package is unable to embed files located
// outside of the same or underlying directory.

//go:embed build
var clientBuildFS embed.FS

func main() {
	home.Main(clientBuildFS)
}
