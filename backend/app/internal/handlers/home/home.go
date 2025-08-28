package home

import (
    "net/http"

    "PulpuVOX/web/templates/pages/home"
    "github.com/markbates/goth"
)

func Handler(w http.ResponseWriter, r *http.Request) {
    // Get user from context (set by auth middleware)
    user, ok := r.Context().Value("user").(*goth.User)
    if !ok {
        user = nil
    }

    w.Header().Set("Content-Type", "text/html")
    home.Home(user).Render(r.Context(), w)
}
