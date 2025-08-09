package main

import (
	"os"
	"fmt"
	"net/http"
	"github.com/a-h/templ"
	"PulpuVOX/pages/home"
)

func main() {
	http.Handle("/", http.RedirectHandler("/home", http.StatusSeeOther))
	http.HandleFunc("/home", func(w http.ResponseWriter, r *http.Request) {
		templ.Handler(home.Home()).ServeHTTP(w, r)
	})

	port := os.Getenv("BACKEND_PORT")
	fmt.Printf("Serving on port %s ...\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
