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
	switch r.URL.Path {
	case "/manifest.webmanifest", "/sw.js", "/offline.html":
		file = path.Join("assets", file)
	case "/assets/main.css", "/assets/htmx.min.js", "/assets/icon-192.png", "/assets/icon-512.png":
		file = path.Join("assets", path.Base(r.URL.Path))
	default:
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

	contentType := mime.TypeByExtension(path.Ext(file))
	if path.Ext(file) == ".webmanifest" {
		contentType = "application/manifest+json"
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	// Static assets are immutable for the life of the deployment.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, file, info.ModTime(), f.(io.ReadSeeker))
})
