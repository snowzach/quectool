package embed

import (
	"embed"
	"io/fs"
)

//go:embed public_html
var publicHTML embed.FS

func PublicHTMLFS() fs.FS {
	publicHTMLfs, _ := fs.Sub(publicHTML, "public_html")
	return publicHTMLfs
}
