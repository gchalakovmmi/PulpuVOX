package conversation

import (
    "net/http"

    "PulpuVOX/web/templates/pages/conversation"
    "github.com/markbates/goth"
)

func Handler(w http.ResponseWriter, r *http.Request) {
    // Get user from context (set by auth middleware)
    user, ok := r.Context().Value("user").(*goth.User)
    if !ok {
        user = nil
    }

    w.Header().Set("Content-Type", "text/html")
    conversation.Conversation(user).Render(r.Context(), w)
}
