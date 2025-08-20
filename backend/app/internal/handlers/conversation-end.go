package handlers

import (
    "encoding/json"
    "net/http"
    "github.com/jackc/pgx/v5"
    "context"
    "log"
    "github.com/gchalakovmmi/PulpuWEB/auth"
)

func ConversationEndHandler(w http.ResponseWriter, r *http.Request, conn *pgx.Conn, googleAuth *auth.GoogleAuth) {
    var request struct {
        History []ConversationTurn `json:"history"`
    }

    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Get user ID from session
    userID, err := getUserIdFromSession(r, conn, googleAuth)
    if err != nil {
        log.Printf("Error getting user from session: %v", err)
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    historyJSON, err := json.Marshal(request.History)
    if err != nil {
        http.Error(w, "Failed to marshal history", http.StatusInternalServerError)
        return
    }

    _, err = conn.Exec(context.Background(),
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
        "status": "success",
        "redirect": "/conversation-analysis",
    })
}
