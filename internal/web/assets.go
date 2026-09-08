package web

import (
	"embed"
	"io"
	"mime"
	"net/http"
	"path"
)

//go:embed assets/*
var assetsFS embed.FS

// Assets serves embedded static files from internal/web/assets.
// GET /assets/{file...}, /manifest.webmanifest, and /sw.js are wired here.
var Assets http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// Map well-known PWA paths to their asset file names.
	file := path.Base(r.URL.Path)
	if r.URL.Path == "/manifest.webmanifest" || r.URL.Path == "/sw.js" {
		file = path.Join("assets", file)
	} else if r.URL.Path == "/assets/main.css" || r.URL.Path == "/assets/htmx.min.js" {
		file = path.Join("assets", path.Base(r.URL.Path))
	} else {
		// Only main.css is served for now; anything else is a 404.
		http.NotFound(w, r)
		return
	}

	f, err := assetsFS.Open(file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(file)))
	// Static assets are immutable for the life of the deployment.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, file, info.ModTime(), f.(io.ReadSeeker))
})
