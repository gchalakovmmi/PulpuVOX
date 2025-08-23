package handlers

import (
		"github.com/a-h/templ"
    "PulpuVOX/pages/conversationanalysis"
    "net/http"
    "github.com/jackc/pgx/v5"
    "context"
    "log"
    "github.com/gchalakovmmi/PulpuWEB/auth"
)

func GetLatestConversationHandler(w http.ResponseWriter, r *http.Request, conn *pgx.Conn, googleAuth *auth.GoogleAuth) {
    // Get user ID from session
    userID, err := getUserIdFromSession(r, conn, googleAuth)
    if err != nil {
        log.Printf("Error getting user from session: %v", err)
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    var historyJSON []byte
    err = conn.QueryRow(context.Background(),
        "SELECT history FROM conversations WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1",
        userID,
    ).Scan(&historyJSON)

    if err != nil {
        log.Printf("Error fetching conversation: %v", err)
        http.Error(w, "No conversation found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(historyJSON)
}

func ConversationAnalysisHandler(w http.ResponseWriter, r *http.Request, conn *pgx.Conn, googleAuth *auth.GoogleAuth) {
    session, err := googleAuth.GetSession(r)
    if err != nil {
        http.Redirect(w, r, "/auth/google", http.StatusTemporaryRedirect)
        return
    }
    user := session.User
    templ.Handler(conversationanalysis.ConversationAnalysis(&user)).ServeHTTP(w, r)
}
