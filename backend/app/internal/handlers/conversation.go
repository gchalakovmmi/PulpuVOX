package handlers

import (
	"net/http"
	"github.com/jackc/pgx/v5"
	"github.com/gchalakovmmi/PulpuWEB/auth"
	"github.com/a-h/templ"
	"PulpuVOX/pages/conversation"
)

func ConversationHandler(w http.ResponseWriter, r *http.Request, conn *pgx.Conn, googleAuth *auth.GoogleAuth) {
		session, err := googleAuth.GetSession(r)
		if err != nil {
				http.Redirect(w, r, "/auth/google", http.StatusTemporaryRedirect)
				return
		}
		user := session.User
		
		templ.Handler(conversation.Conversation(&user)).ServeHTTP(w, r)
}
