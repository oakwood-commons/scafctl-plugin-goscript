// Package main is the entry point for the Go script provider plugin.
package main

import (
	"github.com/oakwood-commons/scafctl-plugin-goscript/internal/goscript"

	sdkplugin "github.com/oakwood-commons/scafctl-plugin-sdk/plugin"
)

func main() {
	sdkplugin.Serve(&goscript.Plugin{})
}
