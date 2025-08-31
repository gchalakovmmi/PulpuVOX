package conversation

import (
    "encoding/json"
    "net/http"
    "log"

    "github.com/jackc/pgx/v5"
    "github.com/gchalakovmmi/PulpuWEB/auth"
)

func ConversationEndHandler(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
    // Get user session from context (set by auth middleware)
    session, ok := r.Context().Value("user_session").(*auth.Session)
    if !ok || session == nil {
        http.Error(w, "User not authenticated", http.StatusUnauthorized)
        return
    }
    user := session.User

    var request struct {
        History []ConversationTurn `json:"history"`
    }

    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Get user ID from database
    var userID int
    err := conn.QueryRow(r.Context(), 
        "SELECT id FROM users WHERE provider = $1 AND id_by_provider = $2", 
        user.Provider, user.UserID).Scan(&userID)
    if err != nil {
        log.Printf("Error getting user ID: %v", err)
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    historyJSON, err := json.Marshal(request.History)
    if err != nil {
        http.Error(w, "Failed to marshal history", http.StatusInternalServerError)
        return
    }

    // Save conversation to database
    _, err = conn.Exec(r.Context(),
        "INSERT INTO conversations (user_id, history) VALUES ($1, $2)",
        userID, historyJSON,
    )
    if err != nil {
        log.Printf("Failed to save conversation: %v", err)
        http.Error(w, "Failed to save conversation", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status":   "success",
        "redirect": "/conversation-analysis",
    })
}
