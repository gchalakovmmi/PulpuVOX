package handlers

import (
	"PulpuVOX/internal/db"
	"PulpuVOX/pages/home"
	"net/http"
	"github.com/jackc/pgx/v5"
	"github.com/gchalakovmmi/PulpuWEB/auth"
	"log"
	"github.com/a-h/templ"
	"PulpuVOX/pages/conversation"
)


func HomeHandler(w http.ResponseWriter, r *http.Request, conn *pgx.Conn, googleAuth *auth.GoogleAuth) {
		session, err := googleAuth.GetSession(r)
		if err != nil {
				http.Redirect(w, r, "/auth/google", http.StatusTemporaryRedirect)
				return
		}
		user := session.User
		
		// Check and create or update user if needed
		dbUser, err := db.GetOrCreateUser(conn, user)
		if err != nil {
				log.Printf("User management error: %v", err)
				http.Error(w, "Failed to process user data", http.StatusInternalServerError)
				return
		}
		
		log.Printf("User processed: %s (DB ID: %d)", dbUser.Name, dbUser.ID)
		
		// Render protected content using templ
		templ.Handler(home.Home(&user)).ServeHTTP(w, r)
}

func ConversationHandler(w http.ResponseWriter, r *http.Request, conn *pgx.Conn, googleAuth *auth.GoogleAuth) {
		session, err := googleAuth.GetSession(r)
		if err != nil {
				http.Redirect(w, r, "/auth/google", http.StatusTemporaryRedirect)
				return
		}
		user := session.User
		
		templ.Handler(conversation.Conversation(&user)).ServeHTTP(w, r)
}
