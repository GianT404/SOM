package main

import (
	"embed"
	"io/fs"
	"net/http"
)

var docsFS embed.FS

func docsHandler() http.Handler {
	sub, err := fs.Sub(docsFS, "docs")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
