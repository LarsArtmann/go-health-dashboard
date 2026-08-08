package dashboard

import (
	"embed"
	"net/http"
)

//go:embed favicon.svg
var faviconFS embed.FS

// FaviconHandler returns an http.HandlerFunc that serves the dashboard
// favicon as an SVG image. Register it at your favicon route.
func (d *Dashboard) FaviconHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		data, err := faviconFS.ReadFile("favicon.svg")
		if err != nil {
			http.Error(w, "dashboard: favicon not found", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		_, _ = w.Write(data)
	}
}
