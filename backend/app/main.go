package main

import (
	"os"
	"fmt"
	"log"
	"net/http"
	"PulpuVOX/pages/home"
	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5"
	"github.com/gchalakovmmi/PulpuWEB/db"

	"time"
	"io"
	"github.com/gchalakovmmi/PulpuWEB/auth"
	"context"
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
		http.ServeFile(w, r, "favicon.ico")
	})
	http.Handle("/", http.RedirectHandler("/home", http.StatusSeeOther))
	http.HandleFunc("/home", func(w http.ResponseWriter, r *http.Request) {
		templ.Handler(home.Home()).ServeHTTP(w, r)
	})
	http.HandleFunc("/db-example", db.WithDB(dbConnectionDetails, func(w http.ResponseWriter, r *http.Request, conn *pgx.Conn){
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var fld string
		err := conn.QueryRow(ctx, "select 'Hello World!' as fld").Scan(&fld)
		if err != nil {
				log.Println("Example query failed. Error:\n%v")
				http.Error(w, "Database error", http.StatusInternalServerError)
		}
		fmt.Println(fld)
		templ.Handler(home.Home()).ServeHTTP(w, r)
	}))
	// Authentication routes
	http.HandleFunc("/auth/google", func(w http.ResponseWriter, r *http.Request) {
		if _, err := googleAuth.GetSession(r); err == nil {
			http.Redirect(w, r, "/protected", http.StatusSeeOther)
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

		http.Redirect(w, r, "/protected", http.StatusSeeOther)
	})

	http.HandleFunc("/logout/google", func(w http.ResponseWriter, r *http.Request) {
		googleAuth.LogoutHandler(w, r)
		googleAuth.ClearSession(w)
		http.Redirect(w, r, "/home", http.StatusTemporaryRedirect)
	})

	// Protected route
	http.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		session, err := googleAuth.GetSession(r)
		if err != nil {
			http.Redirect(w, r, "/auth/google", http.StatusTemporaryRedirect)
			return
		}

		// Render protected content using templ
		comp := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			user := session.User
				_, err := io.WriteString(w, `
<html>
<head><title>User Info</title></head>
<body>
	<img src="`+user.AvatarURL+`" width="80">
	<pre>
Name:          `+user.Name+`
Email:         `+user.Email+`
NickName:      `+user.NickName+`
Location:      `+user.Location+`
Description:   `+user.Description+`
UserID:        `+user.UserID+`
Provider:      `+user.Provider+`
AccessToken:   `+user.AccessToken+`
RefreshToken:  `+user.RefreshToken+`
ExpiresAt:     `+user.ExpiresAt.Format("2006-01-02 15:04")+`
RawData:       `+fmt.Sprint(user.RawData)+`
	</pre>
	<a href="/logout/google">Logout</a>
</body>
</html>`)
				return err
		})

		templ.Handler(comp).ServeHTTP(w, r)
	})

	port := os.Getenv("BACKEND_PORT")
	fmt.Printf("Serving on port %s ...\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
