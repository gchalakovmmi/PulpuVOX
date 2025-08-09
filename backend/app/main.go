package main

import (
	"os"
	"fmt"
	"net/http"
	"PulpuVOX/pages/home"
	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5"
	"github.com/gchalakovmmi/handlers"
)

func main() {
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "favicon.ico")
	})
	http.Handle("/", http.RedirectHandler("/home", http.StatusSeeOther))
	http.HandleFunc("/home", func(w http.ResponseWriter, r *http.Request) {
		templ.Handler(home.Home()).ServeHTTP(w, r)
	})
	http.HandleFunc("/test", handlers.WithDB(func(w http.ResponseWriter, r *http.Request, conn *pgx.Conn){
		templ.Handler(home.Home()).ServeHTTP(w, r)
	}))

	port := os.Getenv("BACKEND_PORT")
	fmt.Printf("Serving on port %s ...\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
