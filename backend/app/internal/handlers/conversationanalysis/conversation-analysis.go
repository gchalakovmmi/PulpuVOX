package conversationanalysis

import (
    "log"
    "net/http"

    "PulpuVOX/web/templates/pages/conversationanalysis"
    "github.com/jackc/pgx/v5"
    "github.com/markbates/goth"
)

func GetLatestConversationHandler(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
    // Get user from context
    user, ok := r.Context().Value("user").(*goth.User)
    if !ok {
        http.Error(w, "User not authenticated", http.StatusUnauthorized)
        return
    }

    // Get internal user ID
    var userID int
    err := conn.QueryRow(r.Context(), "SELECT id FROM users WHERE provider = $1 AND id_by_provider = $2", user.Provider, user.UserID).Scan(&userID)
    if err != nil {
        log.Printf("Error getting user ID: %v", err)
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    var historyJSON []byte
    err = conn.QueryRow(r.Context(),
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

func Handler(w http.ResponseWriter, r *http.Request) {
    // Get user from context (set by auth middleware)
    user, ok := r.Context().Value("user").(*goth.User)
    if !ok {
        user = nil
    }

    w.Header().Set("Content-Type", "text/html")
    conversationanalysis.ConversationAnalysis(user).Render(r.Context(), w)
}
