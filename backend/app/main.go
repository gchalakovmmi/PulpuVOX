package main

import (
	"os"
	"fmt"
	"log"
	"net/http"
	"PulpuVOX/internal/handlers"
	"PulpuVOX/pages/landing"
	"github.com/a-h/templ"
	"github.com/gchalakovmmi/PulpuWEB/db"

	"github.com/gchalakovmmi/PulpuWEB/auth"
	"github.com/jackc/pgx/v5"
)

func main() {
	// Get db connection details
	dbConnectionDetails, err := db.GetPostgresConfig()
	if err != nil {
			log.Fatalf("Failed to get Postgres config: %v", err)
	}

	// Initialize authentication
	authConfig, err := auth.GetGoogleAuthConfig()
	if err != nil {
		log.Fatalf("Error getting Google auth config: %v", err)
	}

	googleAuth := auth.NewGoogleAuth(authConfig)

	// Handle routes
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/logo/favicon.ico")
	})
	http.HandleFunc("/", googleAuth.WithOutGoogleAuth("/home", func(w http.ResponseWriter, r *http.Request) {
		templ.Handler(landing.Landing()).ServeHTTP(w, r)
	}))
	// Authentication routes
	http.HandleFunc("/auth/google", func(w http.ResponseWriter, r *http.Request) {
		if _, err := googleAuth.GetSession(r); err == nil {
			http.Redirect(w, r, "/home", http.StatusSeeOther)
			return
		}
		googleAuth.BeginAuthHandler(w, r)
	})

	http.HandleFunc("/auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		user, err := googleAuth.CompleteUserAuth(w, r)
		if err != nil {
			http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := googleAuth.StoreSession(w, user); err != nil {
			http.Error(w, "Session creation failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/home", http.StatusSeeOther)
	})

	http.HandleFunc("/logout/google", func(w http.ResponseWriter, r *http.Request) {
		googleAuth.LogoutHandler(w, r)
		googleAuth.ClearSession(w)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/home", 
		googleAuth.WithGoogleAuth(
			db.WithDB(dbConnectionDetails, func(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
				handlers.HomeHandler(w, r, conn, googleAuth)
			}),
		),
	)

	http.HandleFunc("/conversation", 
		googleAuth.WithGoogleAuth(
			db.WithDB(dbConnectionDetails, func(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
				handlers.ConversationHandler(w, r, conn, googleAuth)
			}),
		),
	)

	port := os.Getenv("BACKEND_PORT")
	fmt.Printf("Serving on port %s ...\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
