package main

import (
	"os"
	"fmt"
	"strconv"
	"net/http"
	"PulpuVOX/pages/home"
	"github.com/a-h/templ"
	// "github.com/jackc/pgx/v5"
	"github.com/gchalakovmmi/handlers"
	// "time"
	// "context"
)

func main() {
	dbConnectionDetails := handlers.ConnectionDetails{
		User: 		os.Getenv("POSTGRES_USER"),
		Password:	os.Getenv("POSTGRES_PASSWORD"),
		ServerIP:	os.Getenv("POSTGRES_CONTAINER_NAME"),
		Schema:		os.Getenv("POSTGRES_DB"),
	}
	var err error
	dbConnectionDetails.Port, err = strconv.Atoi(os.Getenv("POSTGRES_PORT"))
	if err != nil {
		panic(fmt.Sprintf("Invalid POSTGRES_PORT: %v", err))
	}

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "favicon.ico")
	})
	http.Handle("/", http.RedirectHandler("/home", http.StatusSeeOther))
	http.HandleFunc("/home", func(w http.ResponseWriter, r *http.Request) {
		templ.Handler(home.Home()).ServeHTTP(w, r)
	})
	// http.HandleFunc("/test", handlers.WithDB(dbConnectionDetails, func(w http.ResponseWriter, r *http.Request, conn *pgx.Conn){
	// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//    defer cancel()
	// 	var fld string
	// 	err := conn.QueryRow(ctx, "select fld from tbl").Scan(&fld)
	// 	if err != nil {
	// 	}
	// 	fmt.Println(fld)
	// 	templ.Handler(home.Home()).ServeHTTP(w, r)
	// }))

	port := os.Getenv("BACKEND_PORT")
	fmt.Printf("Serving on port %s ...\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
