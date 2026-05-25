package server

import (
	"net/http"
	"strings"

	"github.com/QA-Capsule/qa-capsule-community/pkg/core"
)

func staticWebHandler() http.Handler {
	fs := http.FileServer(http.Dir("./web"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || path == "/index.html" || strings.HasSuffix(path, ".html") {
			w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		} else if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") {
			w.Header().Set("Cache-Control", "public, max-age=300")
		}
		w.Header().Set("X-QA-Capsule-Edition", core.EditionID())
		fs.ServeHTTP(w, r)
	})
}
