package main

import (
	"embed"
	"os"

	"github.com/ca-x/nowledge-mem-snap/internal/app"
)

//go:embed all:web/dist
var webAssets embed.FS

func main() {
	os.Exit(app.Run(webAssets, os.Args))
}
