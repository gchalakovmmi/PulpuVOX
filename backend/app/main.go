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

	"time"
	"io"
	"PulpuVOX/authentication"
	"context"
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

	/////////////// Authentication ////////////////////
	// Initialize authentication
	sessionDuration, _ := time.ParseDuration(os.Getenv("SESSION_DURATION"))
	if sessionDuration == 0 {
		sessionDuration = 24 * time.Hour
	}

	authConfig := &authentication.Config{
		GoogleKey:       os.Getenv("GOOGLE_KEY"),
		GoogleSecret:    os.Getenv("GOOGLE_SECRET"),
		// Use the actual domain and port your app is running on
		CallbackURL:     "http://" + os.Getenv("DOMAIN") + "/auth/google/callback",
		SecretKey:       []byte(os.Getenv("SESSION_SECRET")),
		SessionDuration: sessionDuration,
	}

	googleAuth := authentication.NewGoogleAuth(authConfig)

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
			_, err := io.WriteString(w, `<!DOCTYPE html>
<html>
<head>
	<title>Protected Page</title>
	<meta charset="UTF-8">
	<meta name="referrer" content="no-referrer-when-downgrade">
</head>
<body>
	<h1>Protected Content</h1>
	<p><a href="/logout/google">logout</a></p>
	
	<div>
		<h2>User Profile</h2>
		<p>Name: `+user.Name+`</p>
		<p>Email: `+user.Email+`</p>
		<p><img src="`+user.AvatarURL+`" width="50" referrerpolicy="no-referrer"></p>
	</div>
	
	<div>
		<h2>Session Information</h2>
		<p>UserID: `+user.UserID+`</p>
		<p>ExpiresAt: `+user.ExpiresAt.Format(time.RFC3339)+`</p>
	</div>
</body>
</html>`)
			return err
		})

		templ.Handler(comp).ServeHTTP(w, r)
	})
	/////////////// Authentication ////////////////////

	port := os.Getenv("BACKEND_PORT")
	fmt.Printf("Serving on port %s ...\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}
